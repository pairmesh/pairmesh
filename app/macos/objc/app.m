#import <Cocoa/Cocoa.h>

#if __MAC_OS_X_VERSION_MIN_REQUIRED < 101400

#ifndef NSControlStateValueOff
#define NSControlStateValueOff NSOffState
#endif

#ifndef NSControlStateValueOn
#define NSControlStateValueOn NSOnState
#endif

#endif

Boolean runProcessAsAdministrator(NSString *scriptPath, NSArray *arguments,
                                  BOOL isAdmin, NSString **output,
                                  NSString **errorDescription) {
  NSString *allArgs = [arguments componentsJoinedByString:@" "];
  NSString *isAdminPre = @"";
  if (isAdmin) {
    isAdminPre = @"with administrator privileges";
  }
  NSString *fullScript =
      [NSString stringWithFormat:@"%@ %@", scriptPath, allArgs];
  NSDictionary *errorInfo = [NSDictionary new];
  NSString *script = [NSString
      stringWithFormat:@"do shell script \"%@\" %@", fullScript, isAdminPre];
  NSLog(@"script = %@", script);
  NSAppleScript *appleScript = [[NSAppleScript new] initWithSource:script];
  NSAppleEventDescriptor *eventResult =
      [appleScript executeAndReturnError:&errorInfo];
  // Check errorInfo/var/tmp
  if (!eventResult) {
    // Describe common errors
    *errorDescription = nil;
    if ([errorInfo valueForKey:NSAppleScriptErrorNumber]) {
      NSNumber *errorNumber =
          (NSNumber *)[errorInfo valueForKey:NSAppleScriptErrorNumber];
      if ([errorNumber intValue] == -128)
        *errorDescription =
            @"The administrator password is required to do this.";
    }
    // Set error message from provided message
    if (*errorDescription == nil) {
      if ([errorInfo valueForKey:NSAppleScriptErrorMessage])
        *errorDescription =
            (NSString *)[errorInfo valueForKey:NSAppleScriptErrorMessage];
    }
    return NO;
  } else {
    // Set output to the AppleScript's output
    *output = [eventResult stringValue];
    return YES;
  }
}

void RunNativeApp(char *path) {
  NSString *binpath = [[NSString alloc] initWithCString:path
                                               encoding:NSUTF8StringEncoding];
  free(path);
  NSString *output = nil;
  NSString *errorDescription = nil;
  runProcessAsAdministrator(binpath, @[], YES, &output, &errorDescription);
}
