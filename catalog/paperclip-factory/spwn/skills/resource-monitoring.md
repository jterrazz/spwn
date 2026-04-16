# Resource Monitoring

Use this skill when tracking system resource usage, detecting waste, or
setting up observability for a running process.

## Instructions

1. Identify the resources that matter for this workload: CPU, memory,
   disk I/O, network, open file handles, or process count.
2. Capture a snapshot using standard tools (`top`, `df`, `du`, `ps`,
   `free`, `iostat`). Record timestamps with every measurement.
3. Establish a baseline: what is "normal" for this system under typical
   load? Note the idle state and the peak state.
4. Set thresholds for alerts. A good starting point: warn at 70% of
   capacity, critical at 90%.
5. Look for leaks: is any resource growing monotonically over time?
   Memory that only goes up, disk that only fills, handles that never
   close -- these are the silent killers.
6. Report findings as a table: resource, current value, baseline value,
   trend (stable / growing / shrinking), and recommended action.
7. If a resource is critically low, flag it immediately and suggest a
   concrete remediation step.
