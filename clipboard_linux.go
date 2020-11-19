// Copyright 2021 The golang.design Initiative authors.
// All rights reserved. Use of this source code is governed
// by a GNU GPL-3 license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

// +build linux,!darwin

package clipboard

/*
#cgo LDFLAGS: -lX11
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <stdatomic.h>

int clipboard_test();
int clipboard_write(
	char*          typ,
	unsigned char* buf,
	size_t         n,
	int*           start // FIXME: should use atomic
);
unsigned long clipboard_read(char* typ, char **out);
*/
import "C"
import (
	"fmt"
	"os"
	"runtime"
	"unsafe"
)

func init() {
	ok := C.clipboard_test()
	if ok < 0 {
		panic(`cannot use this package, failed to initialize x11 display, maybe try install:

	apt install -y libx11-dev
`)
	}
}

func read(t MIMEType) (buf []byte) {
	switch t {
	case MIMEText:
		return readc("UTF8_STRING")
	case MIMEImage:
		return readc("image/png")
	}
	return nil
}

func readc(t string) []byte {
	ct := C.CString(t)
	defer C.free(unsafe.Pointer(ct))

	var data *C.char
	n := C.clipboard_read(ct, &data)
	if data == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(data))
	if n <= 0 {
		return nil
	}

	return C.GoBytes(unsafe.Pointer(data), C.int(n))
}

func write(t MIMEType, buf []byte) {
	var s string
	switch t {
	case MIMEText:
		s = "UTF8_STRING"
	case MIMEImage:
		s = "image/png"
	}

	var start C.int

	go func() { // surve as a daemon until the ownership is terminated.
		runtime.LockOSThread()
		cs := C.CString(s)
		defer C.free(unsafe.Pointer(cs))

		ok := C.clipboard_write(cs, (*C.uchar)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), &start)
		if ok != C.int(0) {
			fmt.Fprintf(os.Stderr, "write failed with status: %d\n", int(ok))
		}
	}()

	// FIXME: this should race with the code on the C side, start
	// should use an atomic version, and use atomic_load.
	for start == 0 {
	}
	// wait until enter event loop
	return
}