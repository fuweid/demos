// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

// Use BTF
struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 8192);
	__type(key, pid_t);
	__type(value, u64);
} exec_start_btf SEC(".maps");

// Use symbol but it has been deprecated
struct bpf_map_def SEC("maps") exec_start_symbol = {
	.type        = BPF_MAP_TYPE_HASH,
	.key_size    = sizeof(pid_t),
	.value_size  = sizeof(u64),
	.max_entries = 8192,
};

SEC("tp/sched/sched_process_exec")
int handle_exec(struct trace_event_raw_sched_process_exec *ctx)
{
	pid_t pid;
	u64 ts;

	pid = bpf_get_current_pid_tgid() >> 32;
	ts = bpf_ktime_get_ns();

	bpf_map_update_elem(&exec_start_btf, &pid, &ts, BPF_ANY);
	bpf_map_update_elem(&exec_start_symbol, &pid, &ts, BPF_ANY);
	return 0;
}
