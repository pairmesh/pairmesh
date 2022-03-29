# Apple's SMJobBless Demo 1.5

###I have simply edited the README to match the folder structure of Xcode 5.

Apple's original source:

<https://developer.apple.com/library/mac/samplecode/SMJobBless/Introduction/Intro.html>

SMJobBless demonstrates how to embed a privileged helper tool in an application, how to securely install that tool, and how to associate the tool with the application that invokes it.

SMJobBless uses the Service Management framework that was introduced in Mac OS X 10.6 Snow Leopard.  As of 10.6 this is the preferred method for managing privilege escalation on OS X and should be used instead of earlier approaches such as BetterAuthorizationSample or directly calling AuthorizationExecuteWithPrivileges.


## Packing List

The sample contains the following items:

* Read Me.txt -- this file
* SMJobBless.xcodeproj -- an Xcode project for the sample
* SMJobBlessApp -- the source code for the app
* SMJobBlessHelper -- the source code for the privileged helper tool
* SMJobBlessUtil.py -- a tool for debugging and correcting SMJobBless setup issues
* Uninstall.sh -- a script to uninstall the helper while testing

Within the "SMJobBlessApp" you will find:
* SMJobBlessApp-Info.plist -- the application's property list
* MainMenu.xib -- the app's nib
* main.m -- a standard Cocoa main function
* SMJobBlessAppController.{h,m} -- the sources for SMJobBlessApp

Within the "SMJobBlessHelper" you will find:
* SMJobBlessHelper-Info.plist -- the property list for the privileged helper tool
* SMJobBlessHelper-Launchd.plist -- the launchd property list for the privileged helper tool
* SMJobBlessHelper.c -- the source for the privileged helper tool


## Building and Running the Sample

The sample was built using Xcode 4.6 on OS X 10.8.2.

The Service Management framework uses code signatures to ensure that the helper tool is the one expected to be run by the main application. SMJobBless assumes you're using an Apple-issued Developer ID.  If you don't have a Developer ID, you should get one before proceeding.

<https://developer.apple.com/resources/developer-id/>

The project is set up to use whatever Developer ID you have installed.  However, various entries in various Info.plist files reference the specifics of this Developer ID and these have to be changed to reference your Developer ID.  You can do this as follows:

1. Build the sample in the normal way (Product > Build).

2. Run SMJobBlessUtil.py's "setreq" command to make the necessary Info.plist changes:

```shell
$ ./SMJobBlessUtil.py setreq Build/Products/Debug/SMJobBlessApp.app SMJobBlessApp/SMJobBlessApp-Info.plist SMJobBlessHelper/SMJobBlessHelper-Info.plist
```

Expected successful response:

```shell
SMJobBlessApp/SMJobBlessApp-Info.plist: updated
SMJobBlessHelper/SMJobBlessHelper-Info.plist: updated
```

3. Clean the sample (Product > Clean).

4. Build the sample again.

5. Run SMJobBlessUtil.py's "check" command to check that everything is OK.  If there's a problem it prints an informative diagnostic message otherwise; if everything is OK then, as is traditional on UNIX, it prints nothing.

```shell
$ ./SMJobBlessUtil.py check build/Debug/SMJobBlessApp.app
```

6. Run the sample.

Once you run the sample you'll be prompted for an admin user name and password.  Enter your admin user name and password and, if all goes well, the sample's window will show "The Helper Tool is available!" indicating that everything is OK.  If not, you can look in the console log for information about the failure.


## How It Works

This sample shows how to install a privileged helper tool that belongs to an application while addressing these challenges:

1. Preserving the ability to drag-install the application.

2. Operating under the principle of least privilege by isolating privileged code in a separate process instead of having the entire application running with elevated privileges.

3. Avoiding the use of setuid binaries.

4. Requiring the user to authorize the privileged helper tool only once the first time it's used.

5. Ensuring that the tool hasn't been replaced by another potentially malicious tool.

6. Ensuring that the tool hasn't been co-opted by a different potentially malicious application.

Items 1 and 2 are addressed by shipping the helper tool inside the application bundle without requiring any special ownership or permissions.

Items 3 and 4 are addressed by using launchd to run the helper tool as a daemon with root privileges.

Items 4, 5 and 6 are handled by the SMJobBless function.  See the comments in -awakeFromNib and <ServiceManagement/ServiceManagement.h> for details.

### BASIC PROJECT LAYOUT

There are two targets: SMJobBlessApp and com.apple.bsd.SMJobBlessHelper. The application target depends on the helper target. Once the helper is built, it is copied into the application's Contents/Library/LaunchServices directory. Both targets are signed by your Developer ID. 

### PROPERTY LISTS

The application target has a standard Info.plist associated with it. This property list has an additional key, SMPrivilegedExecutables, whose value is a dictionary. Each key in this dictionary is the name of a privileged helper tool contained in the application. The example has one key, "com.apple.bsd.SMJobBlessHelper". This key maps to a code signing identity, specified as a set of code signing requirements. This is the identity of the tool, ensuring that the app installs the correct tool and not some malicious tool that has been put in place of it. The example requirement is:

anchor apple generic and identifier "com.apple.bsd.SMJobBlessHelper" and (certificate leaf[field.1.2.840.113635.100.6.1.9] /* exists */ or certificate 1[field.1.2.840.113635.100.6.2.6] /* exists */ and certificate leaf[field.1.2.840.113635.100.6.1.13] /* exists */ and certificate leaf[subject.OU] = xxxxxxxxxx)

This states that the helper tool's code signing identifier is "com.apple.bsd.SMJobBlessHelper", and it was signed by the "xxxxxxxxxx" Developer ID.  While this looks complex you don't have to get involved in that complexity.  It turns out that, when you sign the helper tool with a Developer ID, Xcode automatically sets the helper tool's designated requirement like this, and that's what you should use for SMPrivilegedExecutables.  Moreover, this is what the "setreq" command shown above does: extracts the designated requirement from the built tool and put it into the app's Info.plist source code.

More information about the code signing requirements language is available in the "Code Signing Guide".

<https://developer.apple.com/library/mac/#documentation/Security/Conceptual/CodeSigningGuide/RequirementLang/RequirementLang.html>

The helper tool, symmetrically, also specifies the code signing requirements of the applications that are allowed to install it.  This goes into its Info.plist as an array whose key is SMAuthorizedClients. The example has a single entry in the array:

anchor apple generic and identifier "com.apple.bsd.SMJobBlessApp" and (certificate leaf[field.1.2.840.113635.100.6.1.9] /* exists */ or certificate 1[field.1.2.840.113635.100.6.2.6] /* exists */ and certificate leaf[field.1.2.840.113635.100.6.1.13] /* exists */ and certificate leaf[subject.OU] = xxxxxxxxxx)

This is the designated requirement of the app and, as with the tool, you can set it correctly using the "setreq" command.

The helper tool also has a launchd property list (see launchd.plist(5) <x-man-page://5/launchd.plist>) associated with it. For these helper tools, Program and ProgramArguments are not specified. The system will fill in these properties when it installs the helper, since the tools are placed in a predetermined location (currently /Library/PrivilegedHelperTools). So this example has only the Label property.  It's also valid to include other launchd.plist keys; see the man page for details.

The application's Info.plist is distributed in the usual way, by being placed in the bundle.  The helper tool's property lists must be embedded in the executable itself.  This is accomplished by setting special linker flags that create two new sections in the binary: __TEXT,__info_plist and __TEXT,__launchd_plist, for the Info.plist and launchd.plist, respectively. These sections are filled in with the bytes contained in the property list files at link-time.  See the helper tool's build settings, under "Other Linker Flags", for the specific arguments used.

The name of the executable produced by the helper target is "com.apple.bsd.SMJobBlessHelper". This name must be the same as the Label attribute in the launchd.plist, which in turn must be the same as the key in the application's SMPrivilegedExecutables dictionary.


## Caveats

This sample as it stands does not actually talk to the helper tool. The mechanism you should use to talk to your helper tool depends on your system requirements:

* On OS X 10.8 and later you should use NSXPCConnection.  The "Sandboxing with NSXPCConnection" sample is a good place to start with this.

<https://developer.apple.com/library/mac/#samplecode/SandboxingAndNSXPCConnection/>

* On OS X 10.7 and later you can use XPC directly.

* On older systems you should use UNIX domain sockets, as shown by BetterAuthorizationSample.

<https://developer.apple.com/legacy/mac/library/#samplecode/BetterAuthorizationSample/>

The application does not use ARC because it runs on Mac OS X 10.6, which supports on 32-bit hardware, and ARC does not support the 32-bit Objective-C runtime.


## Version History

If you find any problems with this sample, please file a bug against it.

<http://developer.apple.com/bugreporter/>

1.5 (Aug 2013) Set SKIP_INSTALL on the helper tool target so that the app archives properly (r. 14843533).
1.4 (Mar 2013) Updated to use Developer ID (r. 12167747).  Added a tool to help debug problems.  Other minor changes.
1.2 (May 2012) Minor changes to build and run on OS X Mountain Lion
1.1 (Jun 2010) Minor stylistic revisions.
1.0 (Jun 2010) New sample.

Apple Developer Technical Support
Core OS/Hardware

27 Aug 2013
