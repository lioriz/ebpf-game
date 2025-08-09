#include <linux/bpf.h>
#include <linux/ptrace.h>
#include <linux/sched.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

typedef unsigned int u32;
typedef unsigned long long u64;

// Event type identifiers
#define EVT_READ  1
#define EVT_WRITE 2

struct data_t {
    u32 pid;
    u32 event_type;
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

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, u32);
    __type(value, u32);
} target_pids SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1);
    __type(key, u32);
    __type(value, u32);
} print_all_flag SEC(".maps");

int handle_sys_call(struct pt_regs *ctx, u32 event_type)
{
    u32 pid = bpf_get_current_pid_tgid() >> 32;

    // Skip self
    u32 *skip_val = bpf_map_lookup_elem(&skip_pid, &pid);
    if (skip_val) {
        return 0;
    }

    // Check print_all flag
    u32 flag_key = 0;
    u32 *print_all = bpf_map_lookup_elem(&print_all_flag, &flag_key);

    if (print_all && *print_all != 1) {
        // Only allow if PID is targeted
        u32 *target_val = bpf_map_lookup_elem(&target_pids, &pid);
        if (!target_val) {
            return 0;
        }
    }

    struct data_t data = {};
    data.pid = pid;
    data.event_type = event_type;
    int ret = bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &data, sizeof(data));
    if (ret) {
        // optional: inc_dropped();
        return 0;
    }
    return 0;
}

SEC("kprobe/sys_read")
int sys_read_call(struct pt_regs *ctx)
{
    return handle_sys_call(ctx, EVT_READ);
}

SEC("kprobe/sys_read")
int sys_write_call(struct pt_regs *ctx)
{
    return handle_sys_call(ctx, EVT_WRITE);
}

char _license[] SEC("license") = "GPL";
