use super::Reason;

#[test]
fn each_reason_the_menu_bar_numbers_is_read_and_any_other_is_the_owners_quit() {
    // Given - the four numbers the C entry points carry, and two they never send
    let codes = [0, 1, 2, 3, 4, -1];

    // When - each is read
    let reasons: Vec<Reason> = codes.iter().map(|code| Reason::of(*code)).collect();

    // Then - user, logout, restart, shutdown, then the owner's quit for the strangers
    assert_eq!(
        reasons,
        [
            Reason::User,
            Reason::Logout,
            Reason::Restart,
            Reason::Shutdown,
            Reason::User,
            Reason::User
        ]
    );
    assert_eq!(
        reasons.iter().map(|reason| reason.word()).collect::<Vec<_>>(),
        ["user", "logout", "restart", "shutdown", "user", "user"]
    );
}
