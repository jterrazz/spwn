#!/bin/bash
set -u
SELF=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")
ROUTES=${GATE_ROUTES_DIR:-$HOME/.config/spwn/gate-routes}
refuse() { echo "gate-ssh: $*" >&2; exit 92; }
usage() { echo "usage: gate-ssh.sh ssh|rsh [-n] [-q] [-o ConnectTimeout=N] <host> [command...]; gate-ssh.sh scp [-q] <source>... <destination>" >&2; exit 2; }
# invariant: the three options below lead every argv; the first value ssh reads for an option wins, so nothing a caller or ~/.ssh/config says can reach an ssh agent or a default identity.
GUARD=(-o BatchMode=yes -o IdentitiesOnly=yes -o IdentityAgent=none)

expand() { case $1 in "~/"*) echo "$HOME/${1#\~/}" ;; *) echo "$1" ;; esac; }

load_route() {
  local host=$1 f line perm
  case $host in "" | *[!A-Za-z0-9._-]*) refuse "'$host' is not a plain host name: name the gate host with GATE_VM_LINUX" ;; esac
  f=$ROUTES/$host.env
  [ -f "$f" ] || refuse "no route for $host: set GATE_ROUTES_DIR to the folder that holds $host.env (KEY=<identity file>, or HOP=<user@host> and HOP_KEY=<identity file> for a two-hop route); this script never falls back to plain ssh"
  perm=$(ls -ld "$f" | cut -c1-10)
  [ "${perm:5:1}" = - ] && [ "${perm:8:1}" = - ] || refuse "route file $f is writable by others: chmod go-w it"
  KEY=; HOP=; HOP_KEY=
  while IFS= read -r line || [ -n "$line" ]; do
    case $line in
      "" | "#"*) ;;
      KEY=*) KEY=$(expand "${line#KEY=}") ;;
      HOP=*) HOP=${line#HOP=} ;;
      HOP_KEY=*) HOP_KEY=$(expand "${line#HOP_KEY=}") ;;
      *) refuse "route file $f holds a line that is not KEY=, HOP= or HOP_KEY=" ;;
    esac
  done < "$f"
  if [ -n "$HOP" ]; then
    [ -z "$KEY" ] && [ -n "$HOP_KEY" ] || refuse "route file $f: a two-hop route sets HOP and HOP_KEY and no KEY"
    case $HOP in "" | -* | *[!A-Za-z0-9._@-]*) refuse "route file $f: HOP is not user@host" ;; esac
    [ -r "$HOP_KEY" ] || refuse "route file $f: HOP_KEY $HOP_KEY is not a readable file"
  else
    [ -n "$KEY" ] && [ -z "$HOP_KEY" ] || refuse "route file $f: a direct route sets KEY and nothing else"
    [ -r "$KEY" ] || refuse "route file $f: KEY $KEY is not a readable file"
    case $KEY in "$HOME"/.ssh/id_*) refuse "route file $f: KEY is a default identity of the owner: a dedicated key only" ;; esac
  fi
}

launch() { if [ -n "${GATE_SSH_DRYRUN:-}" ]; then printf '%s\n' "$@"; exit 0; fi; exec "$@"; }

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
    launch /usr/bin/ssh "${GUARD[@]}" -i "$KEY" ${flags[@]+"${flags[@]}"} "$host" ${rest[@]+"${rest[@]}"}
  fi
  [ ${#rest[@]} -gt 0 ] || { echo "gate-ssh: a two-hop route has no interactive login" >&2; exit 2; }
  cmd="${rest[*]}"
  b=$(printf %s "$cmd" | base64 | tr -d '\n')
  launch /usr/bin/ssh "${GUARD[@]}" -i "$HOP_KEY" ${flags[@]+"${flags[@]}"} "$HOP" \
    "ssh -o BatchMode=yes $host 'eval \"\$(printf %s $b | base64 -d)\"'"
}

scp_mode() {
  local flags=() paths=() host= remote a h m dst srcs s rd d out rs
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
      *:*) h=${a%%:*}; [ -z "$host" ] && host=$h; [ "$h" = "$host" ] || refuse "scp names two hosts: $host and $h" ;;
    esac
  done
  [ -n "$host" ] || usage
  load_route "$host"
  if [ -z "$HOP" ]; then
    launch /usr/bin/scp "${GUARD[@]}" -i "$KEY" ${flags[@]+"${flags[@]}"} "${paths[@]}"
  fi
  m=${#paths[@]}; dst=${paths[$((m-1))]}; srcs=("${paths[@]:0:$((m-1))}")
  if [[ $dst == $host:* ]]; then
    rd=${dst#"$host":}; [ -n "$rd" ] || rd=.
    for s in "${srcs[@]}"; do
      [[ $s != *:* ]] || refuse "scp between two remote paths is not routed"
      [ -f "$s" ] || { echo "gate-ssh: $s: no such file" >&2; exit 1; }
      "$SELF" ssh "$host" "d=$rd; if [ -d \"\$d\" ]; then d=\"\$d/$(basename "$s")\"; fi; cat > \"\$d\"" < "$s" || exit $?
    done
    exit 0
  fi
  [ ${#srcs[@]} -eq 1 ] || { echo "gate-ssh: a two-hop route copies one remote source at a time" >&2; exit 2; }
  rs=${srcs[0]#"$host":}; out=$dst
  [ -d "$out" ] && out="$out/$(basename "$rs")"
  "$SELF" ssh -n "$host" "cat $rs" > "$out" || { rd=$?; rm -f "$out"; exit $rd; }
}

mode=${1:-}; shift || usage
case $mode in
  ssh | rsh) ssh_mode "$@" ;;
  scp) scp_mode "$@" ;;
  *) usage ;;
esac
