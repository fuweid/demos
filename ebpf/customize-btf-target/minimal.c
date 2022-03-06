// SPDX-License-Identifier: (LGPL-2.1 OR BSD-2-Clause)
/* Copyright (c) 2020 Facebook */
//
// Based on https://github.com/libbpf/libbpf-bootstrap/blob/84d69aaf79d222d1b24db58bbc6c85c5558ebe2c/examples/c/minimal.c
#include <stdio.h>
#include <unistd.h>
#include <argp.h>
#include <sys/resource.h>
#include <bpf/libbpf.h>
#include "minimal.skel.h"

static struct env {
	char *target;
	bool verbose;
} env = {};

const char *argp_program_version = "minimal v0.1";
const char argp_program_doc[] =
"DEMO: Load BPF with customized BTF target\n"
"\n"
"USAGE: minimal [--help] [-t CUSTOMIZED_BTF_TARGET]\n"
"\n"
"EXAMPLES:\n"
"    minimal              		# load bpf with CONFIG_DEBUG_INFO_BTF=yes\n"
"    minimal -t /tmp/vmlinux.btf	# load bpf with /tmp/vmlinux.btf as target CORE-RELO\n";

static const struct argp_option argp_opts[] = {
	{ "target", 't', "/sys/kernel/btf/vmlinux", 0, "default BTF path"},
	{ "verbose", 'v', NULL, 0, "Verbose debug output" },
	{ NULL, 'h', NULL, OPTION_HIDDEN, "Show the full help" },
	{},
};

static error_t parse_arg(int key, char *arg, struct argp_state *state)
{
	switch (key) {
	case 'h':
		argp_state_help(state, stderr, ARGP_HELP_STD_HELP);
		break;
	case 'v':
		env.verbose = true;
		break;
	case 't':
		env.target = arg;
		break;
	default:
		return ARGP_ERR_UNKNOWN;
	}
	return 0;
}


static int libbpf_print_fn(enum libbpf_print_level level, const char *format, va_list args)
{
	if (level == LIBBPF_DEBUG && !env.verbose)
		return 0;

	return vfprintf(stderr, format, args);
}

int main(int argc, char **argv)
{
	struct minimal_bpf *skel;
	DECLARE_LIBBPF_OPTS(bpf_object_open_opts, skel_opts);
	static const struct argp argp = {
		.options = argp_opts,
		.parser = parse_arg,
		.doc = argp_program_doc,
	};
	int err;

	err = argp_parse(&argp, argc, argv, 0, NULL, NULL);
	if (err)
		return err;

	libbpf_set_strict_mode(LIBBPF_STRICT_ALL);
	/* Set up libbpf errors and debug info callback */
	libbpf_set_print(libbpf_print_fn);

	skel_opts.btf_custom_path = env.target;
	skel = minimal_bpf__open_opts(&skel_opts);
	if (!skel) {
		fprintf(stderr, "Failed to open BPF skeleton\n");
		return 1;
	}

	/* Load & verify BPF programs */
	err = minimal_bpf__load(skel);
	if (err) {
		fprintf(stderr, "Failed to load and verify BPF skeleton\n");
		goto cleanup;
	}

	/* Attach tracepoint handler */
	err = minimal_bpf__attach(skel);
	if (err) {
		fprintf(stderr, "Failed to attach BPF skeleton\n");
		goto cleanup;
	}

	printf("Successfully started! Please run `sudo cat /sys/kernel/debug/tracing/trace_pipe` "
	       "to see output of the BPF programs.\n");

	for (;;) {
		/* trigger our BPF program */
		fprintf(stderr, ".");
		sleep(1);
	}

cleanup:
	minimal_bpf__destroy(skel);
	return -err;
}
