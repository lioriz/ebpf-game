#include <uapi/linux/ptrace.h>

BPF_PERF_OUTPUT(events);

int sys_read_probe(struct pt_regs *ctx) {
    char msg[] = "hello sys_read was called";
    events.perf_submit(ctx, &msg, sizeof(msg));
    return 0;
}

int sys_write_probe(struct pt_regs *ctx) {
    char msg[] = "hello sys_write was called";
    events.perf_submit(ctx, &msg, sizeof(msg));
    return 0;
} 