#include <uapi/linux/ptrace.h>
#include <linux/sched.h>

BPF_HASH(skip_pid, u32, u32);

int sys_read_enter(struct pt_regs *ctx, int fd, char __user *buf, size_t count)
{
    u32 pid = bpf_get_current_pid_tgid();
    u32 *val = skip_pid.lookup(&pid);
    if (val) {
        // bpf_trace_printk("sys_read skip pid %d\n", *val);
        return 0;
    }
    bpf_trace_printk("hello sys_read was called\n");
    return 0;
}

int sys_write_enter(struct pt_regs *ctx, int fd, const char __user *buf, size_t count)
{
    u32 pid = bpf_get_current_pid_tgid();
    u32 *val = skip_pid.lookup(&pid);
    if (val) {
        // bpf_trace_printk("sys_write skip pid %d\n", *val);
        return 0;
    }
    bpf_trace_printk("hello sys_write was called\n");
    return 0;
}
