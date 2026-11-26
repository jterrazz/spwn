#!/bin/bash
set -u
unset CDPATH
SELF=$(cd "$(command -p dirname "${BASH_SOURCE[0]}")" && pwd -P)/$(command -p basename "${BASH_SOURCE[0]}")
ROUTES=${GATE_ROUTES_DIR:-$HOME/.config/spwn/gate-routes}
refuse() { echo "gate-ssh: $*" >&2; exit 92; }
usage() { echo "usage: gate-ssh.sh ssh|rsh [-n] [-q] [-T] [-o ConnectTimeout=N] <host> [command...]; gate-ssh.sh scp [-q] <source>... <destination>" >&2; exit 2; }
# invariant: -F /dev/null keeps every ssh_config out, so the options below and the route file's are the whole configuration: no agent, no default identity, no proxy or jump the route did not name.
COMMON=(-F /dev/null -o BatchMode=yes -o StrictHostKeyChecking=yes -o UpdateHostKeys=no -o GlobalKnownHostsFile=/dev/null -o IdentitiesOnly=yes -o IdentityAgent=none -o ForwardAgent=no -o ControlMaster=no -o ControlPath=none -o PermitLocalCommand=no -o EscapeChar=none)
SSH_GUARD=(-a "${COMMON[@]}")
ME=$(command -p id -u)

expand() { case $1 in "~/"*) printf '%s\n' "$HOME/${1#\~/}" ;; *) printf '%s\n' "$1" ;; esac; }
plain_path() {
  case $2 in /*) ;; *) refuse "route file $f: $1 $2 is not an absolute path" ;; esac
  case $2 in *[!A-Za-z0-9._/-]*) refuse "route file $f: $1 holds a character outside A-Z a-z 0-9 . _ / -, which ssh would expand (% or \${) after this check" ;; esac
}
resolve() {
  local p=$1 l n=0 d
  while [ -L "$p" ]; do
    n=$((n + 1)); [ $n -le 16 ] || return 1
    l=$(command -p readlink "$p") || return 1
    case $l in /*) p=$l ;; *) p=$(command -p dirname "$p")/$l ;; esac
  done
  d=$(cd -P "$(command -p dirname "$p")" 2> /dev/null && pwd -P) || return 1
  printf '%s\n' "$d/$(command -p basename "$p")"
}
quote() { local q="'" r="'\\''"; printf "'%s'" "${1//$q/$r}"; }
private() {
  local perm uid
  read -r perm _ uid _ <<< "$(command -p ls -ldn "$1" 2> /dev/null)"
  [ "$uid" = "$ME" ] && [ "${perm:5:1}" = - ] && [ "${perm:8:1}" = - ]
}
sound() {
  local perm uid
  read -r perm _ uid _ <<< "$(command -p ls -ldn "$1" 2> /dev/null)"
  { [ "$uid" = "$ME" ] || [ "$uid" = 0 ]; } && [ "${perm:5:1}" = - ] && [ "${perm:8:1}" = - ]
}

secret() {
  local perm uid
  read -r perm _ uid _ <<< "$(command -p ls -ldn "$1" 2> /dev/null)"
  [ "$uid" = "$ME" ] || return 1
  case ${perm:0:10} in -rw------- | -r--------) ;; *) return 1 ;; esac
}

ssh_folders() {
  local u oh
  u=$(command -p id -un)
  case $u in "" | -* | *[!A-Za-z0-9._-]*) refuse "the user name '$u' is not plain: cannot find its home in the passwd database" ;; esac
  eval "oh=~$u"
  case $oh in /*) ;; *) refuse "no home for $u in the passwd database" ;; esac
  SSH_DIRS=("$oh/.ssh")
  [ -z "${HOME:-}" ] || [ "$HOME/.ssh" -ef "$oh/.ssh" ] || SSH_DIRS+=("$HOME/.ssh")
}

outside_ssh() {
  local name=$1 r=$2 d=${2%/*} s x
  while :; do
    for s in "${SSH_DIRS[@]}"; do
      [ ! "$d" -ef "$s" ] || refuse "route file $f: $name $r lies under $s, where the owner's own identities live: $3"
    done
    [ -n "$d" ] || break
    d=${d%/*}
  done
  for s in "${SSH_DIRS[@]}"; do
    [ -d "$s" ] || continue
    x=$(command -p find -L "$s" -samefile "$r" 2> /dev/null | command -p head -n 1)
    [ -z "$x" ] || refuse "route file $f: $name $r is the same file as $x, a file of the owner's ~/.ssh: $3"
  done
}

route_file() {
  local name=$1 p=$2 r
  plain_path "$name" "$p"
  [ ! -L "$p" ] || refuse "route file $f: $name $p is a symbolic link: name the file itself"
  r=$(resolve "$p") || r=$p
  plain_path "$name" "$r"
  outside_ssh "$name" "$r" "$3"
  [ -f "$r" ] || refuse "route file $f: $name $p is not a file"
  sound "${r%/*}" || refuse "route file $f: the folder of $name $r is not owned by you or root, or is writable by someone else"
  RESOLVED=$r
}

check_key() {
  local name=$1 r
  route_file "$name" "$2" "a gate route takes a dedicated key kept outside ~/.ssh, so this route is refused until one exists, and creating and authorising that key is the owner's gesture (this script never creates, copies or moves a key)"
  r=$RESOLVED
  secret "$r" || refuse "route file $f: $name $r must be owned by you and readable by you alone (mode 600 or 400 exactly)"
  [ -r "$r" ] || refuse "route file $f: $name $r is not a readable file"
  ROUTE+=(-o "IdentityFile=$r")
}

check_known_hosts() {
  [ -n "$KNOWN_HOSTS" ] || refuse "route file $f: KNOWN_HOSTS is unset: every route names the host keys it checks, in a file of its own, so no gate connection reads or writes ~/.ssh/known_hosts"
  route_file KNOWN_HOSTS "$KNOWN_HOSTS" "a gate route keeps its host keys outside ~/.ssh"
  private "$RESOLVED" || refuse "route file $f: KNOWN_HOSTS $RESOLVED is not yours alone: it must be owned by you and writable by no one else"
  ROUTE+=(-o "UserKnownHostsFile=$RESOLVED")
}

load_route() {
  local host=$1 line k v r seen=" "
  case $host in "" | -* | *[!A-Za-z0-9._-]*) refuse "'$host' is not a plain host name: name the gate host with GATE_VM_LINUX" ;; esac
  [ -d "$ROUTES" ] || refuse "no route for $host: set GATE_ROUTES_DIR to the folder that holds $host.env; this script never falls back to plain ssh"
  [ ! -L "$ROUTES" ] || refuse "route folder $ROUTES is a symbolic link: name the folder itself"
  private "$ROUTES" || refuse "route folder $ROUTES is not yours alone: it must be owned by you and writable by no one else"
  f=$ROUTES/$host.env
  [ ! -L "$f" ] || refuse "route file $f is a symbolic link: write the file itself"
  [ -f "$f" ] || refuse "no route for $host: set GATE_ROUTES_DIR to the folder that holds $host.env (KNOWN_HOSTS= always; KEY=, HOSTNAME=, USER=, PORT= and PROXY=station-pipe with STATION=; or HOP= and HOP_KEY=); this script never falls back to plain ssh"
  private "$f" || refuse "route file $f is not yours alone: it must be owned by you and writable by no one else"
  KEY=; HOSTNAME_=; USER_=; PORT=; PROXY=; STATION=; KNOWN_HOSTS=; HOP=; HOP_KEY=
  while IFS= read -r line || [ -n "$line" ]; do
    case $line in "" | "#"*) continue ;; esac
    k=${line%%=*}; v=${line#*=}
    [ "$k" != "$line" ] || refuse "route file $f holds a line that is not KEY=value"
    case $seen in *" $k "*) refuse "route file $f sets $k twice" ;; esac
    seen="$seen$k "
    case $k in
      KEY) KEY=$(expand "$v") ;;
      HOSTNAME) HOSTNAME_=$v ;;
      USER) USER_=$v ;;
      PORT) PORT=$v ;;
      PROXY) PROXY=$v ;;
      STATION) STATION=$v ;;
      HOP) HOP=$v ;;
      KNOWN_HOSTS) KNOWN_HOSTS=$(expand "$v") ;;
      HOP_KEY) HOP_KEY=$(expand "$v") ;;
      *) refuse "route file $f holds $k, which is none of KEY, HOSTNAME, USER, PORT, PROXY, STATION, KNOWN_HOSTS, HOP, HOP_KEY" ;;
    esac
  done < "$f"
  ssh_folders
  ROUTE=()
  if [ -n "$HOP" ]; then
    [ -z "$KEY$HOSTNAME_$USER_$PORT$PROXY$STATION" ] && [ -n "$HOP_KEY" ] || refuse "route file $f: a two-hop route sets HOP, HOP_KEY and KNOWN_HOSTS and nothing else"
    case $HOP in *@*@* | -* | *[!A-Za-z0-9._@-]*) refuse "route file $f: HOP is not user@host" ;; esac
    case $HOP in [A-Za-z0-9._]*@[A-Za-z0-9._]*) ;; *) refuse "route file $f: HOP is not user@host" ;; esac
    check_key HOP_KEY "$HOP_KEY"
    check_known_hosts
    return
  fi
  [ -n "$KEY" ] && [ -z "$HOP_KEY" ] || refuse "route file $f: a direct route sets KEY, and HOP_KEY only with HOP"
  check_key KEY "$KEY"
  check_known_hosts
  if [ -n "$HOSTNAME_" ]; then
    case $HOSTNAME_ in -* | *[!A-Za-z0-9._-]*) refuse "route file $f: HOSTNAME is not a plain host name" ;; esac
    ROUTE+=(-o "HostName=$HOSTNAME_")
  fi
  if [ -n "$USER_" ]; then
    case $USER_ in -* | *[!A-Za-z0-9._-]*) refuse "route file $f: USER is not a plain user name" ;; esac
    ROUTE+=(-o "User=$USER_")
  fi
  if [ -n "$PORT" ]; then
    case $PORT in "" | 0* | *[!0-9]*) refuse "route file $f: PORT is not a port number" ;; esac
    [ "$PORT" -le 65535 ] || refuse "route file $f: PORT is not a port number"
    ROUTE+=(-o "Port=$PORT")
  fi
  case $PROXY in
    "") [ -z "$STATION" ] || refuse "route file $f: STATION is read only with PROXY=station-pipe" ;;
    station-pipe)
      case $STATION in /*) ;; *) refuse "route file $f: PROXY=station-pipe needs STATION=<absolute path of the station binary>" ;; esac
      case $STATION in *[!A-Za-z0-9._/-]*) refuse "route file $f: STATION holds a character outside A-Z a-z 0-9 . _ / -" ;; esac
      r=$(resolve "$STATION") || refuse "route file $f: STATION $STATION is not an executable file"
      case $r in
        */station | */Spwn.app/Contents/MacOS/"Spwn Engine") ;;
        *) refuse "route file $f: STATION $STATION is $r, which is neither a station binary nor Spwn.app's Spwn Engine, the two names the installer ships" ;;
      esac
      case ${r%/*} in *[!A-Za-z0-9._/-]*) refuse "route file $f: STATION resolves to $r, whose folder holds a character outside A-Z a-z 0-9 . _ / -" ;; esac
      outside_ssh STATION "$r" "the proxy is the binary the installer ships, never a file of ~/.ssh"
      [ -f "$r" ] && [ -x "$r" ] || refuse "route file $f: STATION $STATION is not an executable file"
      sound "$r" && sound "${r%/*}" || refuse "route file $f: STATION $r or its folder is not owned by you or root, or is writable by someone else"
      ROUTE+=(-o "ProxyCommand='$r' machines pipe $host")
      ;;
    *) refuse "route file $f: PROXY is station-pipe or unset" ;;
  esac
}

launch() {
  local keep=(PATH=/usr/bin:/bin:/usr/sbin:/sbin)
  [ -z "${HOME:-}" ] || keep+=("HOME=$HOME")
  [ -z "${LANG:-}" ] || keep+=("LANG=$LANG")
  set -- /usr/bin/env -i "${keep[@]}" "$@"
  if [ -n "${GATE_SSH_DRYRUN:-}" ]; then printf '%s\n' "$@"; exit 0; fi
  exec "$@"
}

ssh_mode() {
  local flags=() host= rest=() b cmd
  while [ $# -gt 0 ]; do
    case $1 in
      -n | -q | -T) flags+=("$1"); shift ;;
      -o) case ${2:-} in ConnectTimeout=* | ServerAliveInterval=* | ServerAliveCountMax=* | BatchMode=yes) flags+=(-o "$2"); shift 2 ;; *) usage ;; esac ;;
      -*) usage ;;
      *) host=$1; shift; rest=("$@"); break ;;
    esac
  done
  [ -n "$host" ] || usage
  load_route "$host"
  if [ -z "$HOP" ]; then
    launch /usr/bin/ssh "${SSH_GUARD[@]}" ${ROUTE[@]+"${ROUTE[@]}"} ${flags[@]+"${flags[@]}"} -- "$host" ${rest[@]+"${rest[@]}"}
  fi
  [ ${#rest[@]} -gt 0 ] || { echo "gate-ssh: a two-hop route has no interactive login" >&2; exit 2; }
  cmd="${rest[*]}"
  b=$(printf %s "$cmd" | command -p base64 | command -p tr -d '\n')
  launch /usr/bin/ssh "${SSH_GUARD[@]}" ${ROUTE[@]+"${ROUTE[@]}"} ${flags[@]+"${flags[@]}"} -- "$HOP" \
    "ssh -o BatchMode=yes -- $host 'eval \"\$(printf %s $b | base64 -d)\"'"
}

scp_mode() {
  local flags=() paths=() host= a h m dst srcs s rd out rs rc remotes=0
  while [ $# -gt 0 ]; do
    case $1 in
      -q) flags+=("$1"); shift ;;
      -*) usage ;;
      *) paths+=("$1"); shift ;;
    esac
  done
  [ ${#paths[@]} -ge 2 ] || usage
  for a in "${paths[@]}"; do
    case $a in
      *://*) refuse "scp names a URI, $a: name <host>:<path>, so the route says how the host is reached" ;;
      *:*) h=${a%%:*}; remotes=$((remotes + 1)); [ -z "$host" ] && host=$h; [ "$h" = "$host" ] || refuse "scp names two hosts: $host and $h" ;;
    esac
  done
  [ -n "$host" ] || usage
  m=${#paths[@]}; dst=${paths[$((m-1))]}; srcs=("${paths[@]:0:$((m-1))}")
  if [[ $dst == *:* ]]; then [ "$remotes" = 1 ] || refuse "scp between two remote paths is not routed"; fi
  load_route "$host"
  if [ -z "$HOP" ]; then
    launch /usr/bin/scp "${COMMON[@]}" ${ROUTE[@]+"${ROUTE[@]}"} ${flags[@]+"${flags[@]}"} -- "${paths[@]}"
  fi
  if [[ $dst == "$host":* ]]; then
    rd=${dst#"$host":}; [ -n "$rd" ] || rd=.
    for s in "${srcs[@]}"; do
      [ -f "$s" ] || { echo "gate-ssh: $s: no such file" >&2; exit 1; }
      "$BASH" "$SELF" ssh "$host" "d=$(quote "$rd"); if [ -d \"\$d\" ]; then d=\"\$d\"/$(quote "$(command -p basename "$s")"); fi; /bin/cat > \"\$d\"" < "$s" || exit $?
    done
    exit 0
  fi
  [ ${#srcs[@]} -eq 1 ] || { echo "gate-ssh: a two-hop route copies one remote source at a time" >&2; exit 2; }
  rs=${srcs[0]#"$host":}; out=$dst
  [ -d "$out" ] && out="$out/$(command -p basename "$rs")"
  "$BASH" "$SELF" ssh -n "$host" "/bin/cat -- $(quote "$rs")" > "$out" || { rc=$?; command -p rm -f "$out"; exit $rc; }
}

mode=${1:-}; shift || usage
case $mode in
  ssh | rsh) ssh_mode "$@" ;;
  scp) scp_mode "$@" ;;
  *) usage ;;
esac
