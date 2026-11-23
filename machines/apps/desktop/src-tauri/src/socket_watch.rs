use std::fs;
use std::io;
use std::os::unix::fs::MetadataExt;
use std::path::Path;
use std::thread::{self, JoinHandle};

// value: what stands at a path: its device and inode, or nothing.
type Standing = Option<(u64, u64)>;

fn standing(path: &Path) -> Standing {
    fs::symlink_metadata(path)
        .ok()
        .map(|meta| (meta.dev(), meta.ino()))
}

// invariant: the folder of `path` is watched from the call on, on a thread of its own, and `changed` is called each
// invariant: time what stands at the path is no longer what stood there: a socket bound anew, removed or replaced.
// invariant: Another entry of the folder tells nothing. The thread ends when the folder goes.
pub fn watch(path: &Path, changed: impl Fn() + Send + 'static) -> io::Result<JoinHandle<()>> {
    let folder = path
        .parent()
        .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidInput, "a socket path has a folder"))?;
    let entries = folder::Entries::of(folder)?;
    let path = path.to_path_buf();
    let mut stood = standing(&path);
    Ok(thread::spawn(move || {
        while let Ok(true) = entries.next() {
            let stands = standing(&path);
            if stands != stood {
                stood = stands;
                changed();
            }
        }
    }))
}

#[cfg(target_os = "macos")]
mod folder {
    use std::ffi::CString;
    use std::io;
    use std::os::fd::{AsRawFd, FromRawFd, OwnedFd};
    use std::os::unix::ffi::OsStrExt;
    use std::path::Path;
    use std::ptr;

    const GONE: u32 = libc::NOTE_DELETE | libc::NOTE_RENAME | libc::NOTE_REVOKE;

    fn owned(fd: libc::c_int) -> io::Result<OwnedFd> {
        if fd < 0 {
            return Err(io::Error::last_os_error());
        }
        Ok(unsafe { OwnedFd::from_raw_fd(fd) })
    }

    // value: a kqueue that hears each change of a folder's entries, and the folder's own end.
    pub struct Entries {
        queue: OwnedFd,
        _folder: OwnedFd,
    }

    impl Entries {
        pub fn of(folder: &Path) -> io::Result<Entries> {
            let name = CString::new(folder.as_os_str().as_bytes())?;
            let opened =
                owned(unsafe { libc::open(name.as_ptr(), libc::O_EVTONLY | libc::O_CLOEXEC) })?;
            let queue = owned(unsafe { libc::kqueue() })?;
            let change = libc::kevent {
                ident: opened.as_raw_fd() as libc::uintptr_t,
                filter: libc::EVFILT_VNODE,
                flags: libc::EV_ADD | libc::EV_CLEAR,
                fflags: libc::NOTE_WRITE | GONE,
                data: 0,
                udata: ptr::null_mut(),
            };
            let set = unsafe {
                libc::kevent(
                    queue.as_raw_fd(),
                    &change,
                    1,
                    ptr::null_mut(),
                    0,
                    ptr::null(),
                )
            };
            if set < 0 {
                return Err(io::Error::last_os_error());
            }
            Ok(Entries {
                queue,
                _folder: opened,
            })
        }

        // value: true on a change of the folder's entries, false once the folder itself went.
        pub fn next(&self) -> io::Result<bool> {
            loop {
                let mut event: libc::kevent = unsafe { std::mem::zeroed() };
                let heard = unsafe {
                    libc::kevent(
                        self.queue.as_raw_fd(),
                        ptr::null(),
                        0,
                        &mut event,
                        1,
                        ptr::null(),
                    )
                };
                if heard < 0 {
                    let error = io::Error::last_os_error();
                    if error.kind() == io::ErrorKind::Interrupted {
                        continue;
                    }
                    return Err(error);
                }
                if heard > 0 {
                    return Ok(event.fflags & GONE == 0);
                }
            }
        }
    }
}

#[cfg(not(target_os = "macos"))]
mod folder {
    use std::io;
    use std::path::Path;

    pub struct Entries;

    impl Entries {
        pub fn of(_folder: &Path) -> io::Result<Entries> {
            Err(io::Error::new(
                io::ErrorKind::Unsupported,
                "a folder is watched on a Mac only",
            ))
        }

        pub fn next(&self) -> io::Result<bool> {
            Ok(false)
        }
    }
}

#[cfg(test)]
#[path = "socket_watch_test.rs"]
mod tests;
