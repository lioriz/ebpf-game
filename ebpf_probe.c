#include <linux/bpf.h>
#include <linux/ptrace.h>
#include <linux/sched.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

typedef unsigned int u32;
typedef unsigned long long u64;

struct data_t {
    u32 pid;
    char msg[32];
};

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(key_size, sizeof(int));
    __uint(value_size, sizeof(u32));
    __uint(max_entries, 1024);
} events SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, u32);
    __type(value, u32);
} skip_pid SEC(".maps");

SEC("kprobe/__x64_sys_read")
int sys_read_call(struct pt_regs *ctx)
{
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u32 *val = bpf_map_lookup_elem(&skip_pid, &pid);
    if (val) {
        return 0;
    }

    struct data_t data = {};
    data.pid = pid;
    __builtin_strncpy(data.msg, "hello sys_read was called", sizeof(data.msg));
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &data, sizeof(data));
    return 0;
}

SEC("kprobe/__x64_sys_write")
int sys_write_call(struct pt_regs *ctx)
{
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u32 *val = bpf_map_lookup_elem(&skip_pid, &pid);
    if (val) {
        return 0;
    }

    struct data_t data = {};
    data.pid = pid;
    __builtin_strncpy(data.msg, "hello sys_write was called", sizeof(data.msg));
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &data, sizeof(data));
    return 0;
}

char _license[] SEC("license") = "GPL";
