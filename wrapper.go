package posix_mq

/*
#cgo LDFLAGS: -lrt

#include <stdlib.h>
#include <signal.h>
#include <fcntl.h>
#include <mqueue.h>

// Expose non-variadic function requires 4 arguments.
mqd_t mq_open4(const char *name, int oflag, int mode, struct mq_attr *attr) {
	return mq_open(name, oflag, mode, attr);
}
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"
)

const (
	O_RDONLY = C.O_RDONLY
	O_WRONLY = C.O_WRONLY
	O_RDWR   = C.O_RDWR

	O_CLOEXEC  = C.O_CLOEXEC
	O_CREAT    = C.O_CREAT
	O_EXCL     = C.O_EXCL
	O_NONBLOCK = C.O_NONBLOCK

	// Based on Linux 3.5+
	MSGSIZE_MAX     = 16777216
	MSGSIZE_DEFAULT = MSGSIZE_MAX
)

var (
	MemoryAllocationError = fmt.Errorf("Memory Allocation Error")
)

type receiveBuffer struct {
	buf  *C.char
	size C.size_t
}

func newReceiveBuffer(bufSize int) (*receiveBuffer, error) {
	buf := (*C.char)(C.malloc(C.size_t(bufSize)))
	if buf == nil {
		return nil, MemoryAllocationError
	}

	return &receiveBuffer{
		buf:  buf,
		size: C.size_t(bufSize),
	}, nil
}

func (rb *receiveBuffer) free() {
	C.free(unsafe.Pointer(rb.buf))
}

func timeToTimespec(t time.Time) C.struct_timespec {
	return C.struct_timespec{
		tv_sec:  C.long(t.Unix()),
		tv_nsec: C.long(t.Nanosecond() % 1000000000),
	}
}

func mq_open(name string, oflag int, mode int, attr *MessageQueueAttribute) (int, error) {
	var cAttr *C.struct_mq_attr
	if attr != nil {
		cAttr = &C.struct_mq_attr{
			mq_flags:   C.long(attr.Flags),
			mq_maxmsg:  C.long(attr.MaxMsg),
			mq_msgsize: C.long(attr.MsgSize),
			mq_curmsgs: C.long(attr.MsgCnt),
		}
	}
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	h, err := C.mq_open4(cName, C.int(oflag), C.int(mode), cAttr)
	if h == -1 {
		return 0, err
	}

	return int(h), nil
}

func mq_send(h int, data []byte, priority uint) error {
	byteStr := *(*string)(unsafe.Pointer(&data))
	cStr := C.CString(byteStr)
	defer C.free(unsafe.Pointer(cStr))
	rv, err := C.mq_send(C.int(h), cStr, C.size_t(len(data)), C.uint(priority))
	if rv == -1 {
		return err
	}

	return nil
}

func mq_timedsend(h int, data []byte, priority uint, t time.Time) error {
	timeSpec := timeToTimespec(t)

	byteStr := *(*string)(unsafe.Pointer(&data))
	cStr := C.CString(byteStr)
	defer C.free(unsafe.Pointer(cStr))
	rv, err := C.mq_timedsend(C.int(h), cStr, C.size_t(len(data)), C.uint(priority), &timeSpec)
	if rv == -1 {
		return err
	}

	return nil
}

func mq_receive(h int, recvBuf *receiveBuffer) ([]byte, uint, error) {
	var msgPrio C.uint

	size, err := C.mq_receive(C.int(h), recvBuf.buf, recvBuf.size, &msgPrio)
	if size == -1 {
		return nil, 0, err
	}

	return C.GoBytes(unsafe.Pointer(recvBuf.buf), C.int(size)), uint(msgPrio), nil
}

func mq_timedreceive(h int, recvBuf *receiveBuffer, t time.Time) ([]byte, uint, error) {
	var (
		msgPrio  C.uint
		timeSpec = timeToTimespec(t)
	)

	size, err := C.mq_timedreceive(C.int(h), recvBuf.buf, recvBuf.size, &msgPrio, &timeSpec)
	if size == -1 {
		return nil, 0, err
	}

	return C.GoBytes(unsafe.Pointer(recvBuf.buf), C.int(size)), uint(msgPrio), nil
}

func mq_notify(h int, sigNo int) error {
	sigEvent := &C.struct_sigevent{
		sigev_notify: C.SIGEV_SIGNAL, // posix_mq supports only signal.
		sigev_signo:  C.int(sigNo),
	}

	rv, err := C.mq_notify(C.int(h), sigEvent)
	if rv == -1 {
		return err
	}

	return nil
}

func mq_close(h int) error {
	rv, err := C.mq_close(C.int(h))
	if rv == -1 {
		return err
	}

	return nil
}

func mq_unlink(name string) error {
	cStr := C.CString(name)
	defer C.free(unsafe.Pointer(cStr))
	rv, err := C.mq_unlink(cStr)
	if rv == -1 {
		return err
	}

	return nil
}

func mq_getattr(h int) (*MessageQueueAttribute, error) {
	var cAttr C.struct_mq_attr
	rv, err := C.mq_getattr(C.int(h), &cAttr)
	if rv == -1 {
		return nil, err
	}

	mqa := &MessageQueueAttribute{
		Flags:   int(cAttr.mq_flags),
		MaxMsg:  int(cAttr.mq_maxmsg),
		MsgSize: int(cAttr.mq_msgsize),
		MsgCnt:  int(cAttr.mq_curmsgs),
	}

	return mqa, nil
}
