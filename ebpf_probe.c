#include <uapi/linux/ptrace.h>
#include <linux/sched.h>

struct data_t {
    u32 pid;
    char msg[32];
};

BPF_PERF_OUTPUT(events);
BPF_HASH(skip_pid, u32, u32);

int sys_read_call(struct pt_regs *ctx, int fd, char __user *buf, size_t count)
{
    u32 pid = bpf_get_current_pid_tgid();
    u32 *val = skip_pid.lookup(&pid);
    if (val) {
        // bpf_trace_printk("sys_read skip pid %d\n", *val);
        return 0;
    }

    struct data_t data = {};
    data.pid = pid;
    __builtin_strncpy(data.msg, "hello sys_read was called", sizeof(data.msg));
    events.perf_submit(ctx, &data, sizeof(data));
    return 0;
}

int sys_write_call(struct pt_regs *ctx, int fd, const char __user *buf, size_t count)
{
    u32 pid = bpf_get_current_pid_tgid();
    u32 *val = skip_pid.lookup(&pid);
    if (val) {
        // bpf_trace_printk("sys_write skip pid %d\n", *val);
        return 0;
    }

    struct data_t data = {};
    data.pid = pid;
    __builtin_strncpy(data.msg, "hello sys_write was called", sizeof(data.msg));
    events.perf_submit(ctx, &data, sizeof(data));
    return 0;
}
