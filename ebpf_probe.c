#include <uapi/linux/ptrace.h>

int sys_read_enter(struct pt_regs *ctx, int fd, char __user *buf, size_t count)
{
    bpf_trace_printk("hello sys_read was called\n");
    return 0;
}

int sys_write_enter(struct pt_regs *ctx, int fd, const char __user *buf, size_t count)
{
    bpf_trace_printk("hello sys_write was called\n");
    return 0;
}
