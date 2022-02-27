// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

SEC("tp/sched/sched_process_exec")
u64 double_ts(u64 ts) {
	return ts + ts;
}

u64 double_ts_in_text(u64 ts) {
	return ts + ts;
}

SEC("tp/sched/sched_process_exec")
int handle_exec(struct trace_event_raw_sched_process_exec *ctx)
{
	u64 ts;

	ts = bpf_ktime_get_ns();
	ts = double_ts(ts);
	ts = double_ts_in_text(ts);
	return ts;
}

