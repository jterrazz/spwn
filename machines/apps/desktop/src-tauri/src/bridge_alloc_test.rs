use std::alloc::{GlobalAlloc, Layout, System};
use std::cell::Cell;
use std::io::Cursor;
use std::sync::Mutex;

use super::{pump, Message, Sink};

struct Counting;

thread_local! {
    static ALLOCATIONS: Cell<usize> = const { Cell::new(0) };
}

fn count() {
    ALLOCATIONS.with(|allocations| allocations.set(allocations.get() + 1));
}

// invariant: it counts the allocations of the thread that asks, so the tests running beside it never count in.
unsafe impl GlobalAlloc for Counting {
    unsafe fn alloc(&self, layout: Layout) -> *mut u8 {
        count();
        System.alloc(layout)
    }

    unsafe fn dealloc(&self, pointer: *mut u8, layout: Layout) {
        System.dealloc(pointer, layout);
    }

    unsafe fn realloc(&self, pointer: *mut u8, layout: Layout, size: usize) -> *mut u8 {
        count();
        System.realloc(pointer, layout, size)
    }
}

#[global_allocator]
static COUNTING: Counting = Counting;

#[derive(Default)]
struct Kept(Mutex<Vec<Message>>);

impl Sink for Kept {
    fn send(&self, message: Message) -> bool {
        self.0.lock().is_ok_and(|mut kept| {
            kept.push(message);
            true
        })
    }
}

fn answer_of(items: usize) -> String {
    let item =
        r#"{"Name":"machine-12","State":"running","Ports":[1,2,3],"Memory":0.25,"Tags":{"a":"b"}}"#;
    let list = vec![item; items].join(",");
    format!("{{\"v\":2,\"id\":\"1\",\"out\":{{\"Items\":[{list}]}}}}\n")
}

fn allocated_by_pumping(line: &str) -> (usize, Vec<Message>) {
    let kept = Kept::default();
    let before = ALLOCATIONS.with(Cell::get);
    let reason = pump(Cursor::new(line.as_bytes()), "1", false, &kept, 64 << 20);
    let after = ALLOCATIONS.with(Cell::get);
    assert_eq!(reason, "answered");
    (after - before, kept.0.into_inner().unwrap_or_default())
}

#[test]
fn an_answer_is_forwarded_as_the_engines_line_and_no_tree_of_values_is_built() {
    // Given - an answer of 4 000 items, some 360 kB on one line
    let line = answer_of(4000);

    // When - its exchange is pumped
    let (allocations, heard) = allocated_by_pumping(&line);

    // Then - the pump allocated a few times, not once per value, and the window hears the line byte for byte
    assert!(line.len() > 300_000);
    assert!(allocations <= 64, "{allocations} allocations for one frame");
    assert_eq!(heard.len(), 1);
    assert_eq!(
        serde_json::to_string(&heard[0]).expect("a message serializes"),
        format!("{{\"frame\":{}}}", line.trim_end())
    );
}
