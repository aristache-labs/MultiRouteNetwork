//go:build windows

package main

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// executableIcon reads an icon resource without launching the executable.
func executableIcon(path string) ([]byte, error) {
	// GDI device contexts are owned by the calling OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	shell := windows.NewLazySystemDLL("shell32.dll")
	user := windows.NewLazySystemDLL("user32.dll")
	gdi := windows.NewLazySystemDLL("gdi32.dll")
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	const size = 96 // Covers 40 CSS pixels at common high-DPI scales.
	var icon uintptr
	count, _, _ := user.NewProc("PrivateExtractIconsW").Call(uintptr(unsafe.Pointer(file)), 0, size, size, uintptr(unsafe.Pointer(&icon)), 0, 1, 0)
	if count == 0 || count == 0xffffffff || icon == 0 {
		count, _, _ = shell.NewProc("ExtractIconExW").Call(uintptr(unsafe.Pointer(file)), 0, uintptr(unsafe.Pointer(&icon)), 0, 1)
	}
	if count == 0 || icon == 0 {
		return nil, errors.New("no executable icon")
	}
	defer user.NewProc("DestroyIcon").Call(icon)
	dc, _, _ := gdi.NewProc("CreateCompatibleDC").Call(0)
	if dc == 0 {
		return nil, errors.New("create icon context")
	}
	defer gdi.NewProc("DeleteDC").Call(dc)
	type bitmapInfo struct {
		Size                         uint32
		Width, Height                int32
		Planes, BitCount             uint16
		Compression, SizeImage       uint32
		XPelsPerMeter, YPelsPerMeter int32
		ClrUsed, ClrImportant        uint32
	}
	info := bitmapInfo{Size: 40, Width: size, Height: -size, Planes: 1, BitCount: 32}
	var bits unsafe.Pointer
	bitmap, _, _ := gdi.NewProc("CreateDIBSection").Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 || bits == nil {
		return nil, errors.New("create icon bitmap")
	}
	defer gdi.NewProc("DeleteObject").Call(bitmap)
	old, _, _ := gdi.NewProc("SelectObject").Call(dc, bitmap)
	defer gdi.NewProc("SelectObject").Call(dc, old)
	// Preserve native alpha for modern icons. Legacy masks use a neutral backdrop.
	pixels := unsafe.Slice((*byte)(bits), size*size*4)
	for i := 0; i < len(pixels); i += 4 {
		pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = 0, 0, 0, 0
	}
	ok, _, _ := user.NewProc("DrawIconEx").Call(dc, 0, 0, icon, size, size, 0, 0, 3)
	if ok == 0 {
		return nil, errors.New("draw executable icon")
	}
	gdi.NewProc("GdiFlush").Call()
	hasAlpha := false
	for i := 3; i < len(pixels); i += 4 {
		if pixels[i] != 0 {
			hasAlpha = true
			break
		}
	}
	if !hasAlpha {
		for i := 0; i < len(pixels); i += 4 {
			pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = 39, 39, 39, 255
		}
		user.NewProc("DrawIconEx").Call(dc, 0, 0, icon, size, size, 0, 0, 3)
		gdi.NewProc("GdiFlush").Call()
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for i := 0; i < len(pixels); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = pixels[i+2], pixels[i+1], pixels[i], pixels[i+3]
		if !hasAlpha {
			img.Pix[i+3] = 255
		}
	}
	var out bytes.Buffer
	err = png.Encode(&out, img)
	return out.Bytes(), err
}
