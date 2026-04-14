# spwn CLI

The spwn CLI manages worlds, agents, and the universe from inside a container.

## Key Commands
```bash
spwn status                        # System status
spwn ls                            # List worlds
spwn agent ls                      # List agents
spwn msg inbox <name>              # Check messages
spwn msg send <to> --from <me> "msg"  # Send message
```

## Agent Identity
Your mind is at `/mind/` - read your purpose, traits, and profile before starting work.
```bash
cat /mind/core/purpose.md
cat /mind/core/profile.md
cat /mind/core/traits.md
```
