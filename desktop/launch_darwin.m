#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>
#import <stdio.h>

int mr_register_daemon(const char *plist, char *errbuf, int errlen) {
	@autoreleasepool {
		NSString *name = [NSString stringWithUTF8String:plist];
		SMAppService *service = [SMAppService daemonServiceWithPlistName:name];
		NSError *err = nil;
		if (![service registerAndReturnError:&err]) {
			NSString *text = err.localizedDescription ?: @"authorization required";
			snprintf(errbuf, (size_t)errlen, "%s", text.UTF8String);
			return 1;
		}
		return 0;
	}
}
