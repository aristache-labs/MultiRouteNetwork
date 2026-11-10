//go:build darwin

package main

/*
#cgo LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>
#import <stdlib.h>
#import <string.h>

int mr_icon_png(const char *path, unsigned char **out, int *outlen) {
	@autoreleasepool {
		NSString *file = [NSString stringWithUTF8String:path];
		NSImage *image = [[NSWorkspace sharedWorkspace] iconForFile:file];
		if (!image)
			return 1;
		NSData *tiff = [image TIFFRepresentation];
		NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:tiff];
		NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
		if (!png)
			return 1;
		*outlen = (int)png.length;
		*out = malloc((size_t)*outlen);
		if (!*out)
			return 1;
		memcpy(*out, png.bytes, (size_t)*outlen);
		return 0;
	}
}
*/
import "C"
import (
	"errors"
	"unsafe"
)

func executableIcon(path string) ([]byte, error) {
	cstr := C.CString(path)
	defer C.free(unsafe.Pointer(cstr))
	var out *C.uchar
	var n C.int
	if C.mr_icon_png(cstr, &out, &n) != 0 || n <= 0 {
		return nil, errors.New("no application icon")
	}
	defer C.free(unsafe.Pointer(out))
	return C.GoBytes(unsafe.Pointer(out), n), nil
}
