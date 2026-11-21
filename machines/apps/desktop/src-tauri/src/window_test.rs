use std::cell::{Cell, RefCell};
use std::time::Duration;

use super::{capped, Shown, HARD_CAP};

#[test]
fn the_window_is_claimed_once() {
    let shown = Shown::default();
    assert!(shown.claim());
    assert!(!shown.claim());
    assert!(!shown.claim());
}

#[test]
fn the_hard_cap_shows_a_window_the_page_never_showed() {
    let shown = Shown::default();
    let waited = RefCell::new(Vec::new());
    let shows = Cell::new(0);
    capped(
        |cap| waited.borrow_mut().push(cap),
        || shown.claim(),
        || shows.set(shows.get() + 1),
    );
    assert_eq!(*waited.borrow(), vec![HARD_CAP]);
    assert_eq!(shows.get(), 1);
}

#[test]
fn the_hard_cap_shows_nothing_once_the_page_showed_it() {
    let shown = Shown::default();
    assert!(shown.claim());
    let shows = Cell::new(0);
    capped(|_| {}, || shown.claim(), || shows.set(shows.get() + 1));
    assert_eq!(shows.get(), 0);
}

#[test]
fn the_hard_cap_comes_after_the_pages_own() {
    assert!(HARD_CAP > Duration::from_millis(800));
    assert!(HARD_CAP <= Duration::from_secs(2));
}
