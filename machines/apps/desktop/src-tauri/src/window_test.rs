use std::cell::{Cell, RefCell};
use std::time::Duration;

use super::{capped, dock, Dock, Shown, HARD_CAP};

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

#[test]
fn the_dock_icon_follows_the_window_unless_the_menu_bar_item_is_hidden_too() {
    // Given - the window shown or hidden, the menu bar item shown or hidden
    let cases = [(true, true), (true, false), (false, true), (false, false)];

    // When - each is decided
    let docks: Vec<Dock> = cases
        .iter()
        .map(|(window, item)| dock(*window, *item))
        .collect();

    // Then - hidden only with the window closed and the item there to bring it back
    assert_eq!(docks, [Dock::Shown, Dock::Shown, Dock::Hidden, Dock::Shown]);
}
