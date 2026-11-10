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
