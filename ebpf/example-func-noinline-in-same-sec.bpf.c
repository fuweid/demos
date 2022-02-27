// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

static __attribute__((noinline))
SEC("tp/sched/sched_process_exec")
u64 double_ts(u64 ts) {
	return ts + ts;
}

SEC("tp/sched/sched_process_exec")
int handle_exec(struct trace_event_raw_sched_process_exec *ctx)
{
	u64 ts;

	ts = bpf_ktime_get_ns();
	ts = double_ts(ts);
	return ts;
}

