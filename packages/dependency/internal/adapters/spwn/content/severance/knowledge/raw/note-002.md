# 10/14 standup notes — infra sync

attendees: me, priya, marcus, (jen out sick)

- db migration still blocked on the read-replica lag issue. marcus
  thinks it's the new index on `events.created_at` — too wide? he's
  rolling it back tonight and re-running the load test fri morning
- priya finished the terraform module for the staging cluster, PR is
  up, waiting on review from security (she pinged them 3 days ago,
  no response — follow up monday)
- the on-call rotation swap for thanksgiving week: marcus takes tues/wed,
  i take thurs/fri, priya has sat/sun. jen confirms mon when she's back.
- alert noise: pagerduty fired 14x overnight for the same flapping disk
  check. marcus muting it until we fix the threshold.

action items:
- [ ] me: chase security on priya's PR by eod monday
- [ ] marcus: roll back index, re-run load test friday
- [ ] priya: document the new terraform module once it lands
- [ ] all: rotation confirmed w/ jen monday

next sync: thursday 10/17, same time
