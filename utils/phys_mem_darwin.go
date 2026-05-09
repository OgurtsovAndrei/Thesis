//go:build darwin
package utils

/*
#include <mach/mach.h>
#include <mach/task.h>

static uint64_t get_rss() {
    struct mach_task_basic_info info;
    mach_msg_type_number_t count = MACH_TASK_BASIC_INFO_COUNT;
    if (task_info(mach_task_self(), MACH_TASK_BASIC_INFO, (task_info_t)&info, &count) == KERN_SUCCESS) {
        return info.resident_size;
    }
    return 0;
}
*/
import "C"

func getPhysicalMemStats() (currentRSS, peakRSS uint64) {
    // macOS doesn't easily expose peak RSS via task_info, but getrusage does.
    // However, our polling monitor will track the peak during specific windows.
    return uint64(C.get_rss()), 0
}
