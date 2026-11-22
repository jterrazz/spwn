use std::cell::{Cell, RefCell};

use super::{read, show, LoginItem, Read, Status, NOT_SWITCHED};

struct Fake {
    status: Cell<Status>,
    after: Status,
    refusal: Option<String>,
    asked: RefCell<Vec<&'static str>>,
}

impl Fake {
    fn new(status: Status, after: Status) -> Self {
        Self {
            status: Cell::new(status),
            after,
            refusal: None,
            asked: RefCell::new(Vec::new()),
        }
    }

    fn answered(&self, asked: &'static str) -> Result<(), String> {
        self.asked.borrow_mut().push(asked);
        match &self.refusal {
            Some(message) => Err(message.clone()),
            None => {
                self.status.set(self.after);
                Ok(())
            }
        }
    }
}

impl LoginItem for Fake {
    fn status(&self) -> Status {
        self.status.get()
    }

    fn register(&self) -> Result<(), String> {
        self.answered("register")
    }

    fn unregister(&self) -> Result<(), String> {
        self.answered("unregister")
    }
}

#[test]
fn each_status_macos_holds_is_read_as_the_switch_draws_it() {
    // Given - a login item in each of the four statuses
    let statuses = [
        Status::Enabled,
        Status::NotRegistered,
        Status::RequiresApproval,
        Status::NotFound,
    ];

    // When - each is read
    let read: Vec<Read> = statuses
        .iter()
        .map(|status| read(&Fake::new(*status, *status)))
        .collect();

    // Then - shown, hidden, waiting for approval, and no switch for a bundle that carries none
    assert_eq!(
        read,
        [Read::Shown, Read::Hidden, Read::Approval, Read::Unsupported]
    );
}

#[test]
fn a_press_registers_or_unregisters_then_answers_the_status_read_back() {
    // Given - a hidden item that macOS then holds for approval, and a shown one
    let hidden = Fake::new(Status::NotRegistered, Status::RequiresApproval);
    let shown = Fake::new(Status::Enabled, Status::NotRegistered);

    // When - the first is shown and the second hidden
    let ends = [show(&hidden, true), show(&shown, false)];

    // Then - the status each reads afterwards, each from its one call
    assert_eq!(ends, [Ok(Read::Approval), Ok(Read::Hidden)]);
    assert_eq!(*hidden.asked.borrow(), ["register"]);
    assert_eq!(*shown.asked.borrow(), ["unregister"]);
}

#[test]
fn a_press_already_as_asked_or_with_no_menu_bar_calls_nothing_and_a_refusal_keeps_macos_words() {
    // Given - a shown item, an approval-held one, one absent, and a hidden one whose register is refused
    let shown = Fake::new(Status::Enabled, Status::Enabled);
    let waiting = Fake::new(Status::RequiresApproval, Status::RequiresApproval);
    let absent = Fake::new(Status::NotFound, Status::NotFound);
    let mut refusing = Fake::new(Status::NotRegistered, Status::Enabled);
    refusing.refusal = Some(String::from(
        "SMAppServiceErrorDomain 1: Operation not permitted",
    ));

    // When - each is shown
    let ends = [
        show(&shown, true),
        show(&waiting, true),
        show(&absent, true),
        show(&refusing, true),
    ];

    // Then - the first three as held with no call, the last refused in macOS's words
    assert_eq!(ends[0], Ok(Read::Shown));
    assert_eq!(ends[1], Ok(Read::Approval));
    assert_eq!(ends[2], Ok(Read::Unsupported));
    let refused = ends[3]
        .as_ref()
        .expect_err("a refused register is a refusal");
    assert_eq!(refused.code, NOT_SWITCHED);
    assert_eq!(
        refused.message,
        "SMAppServiceErrorDomain 1: Operation not permitted"
    );
    assert!(shown.asked.borrow().is_empty() && waiting.asked.borrow().is_empty());
    assert!(absent.asked.borrow().is_empty());
}
