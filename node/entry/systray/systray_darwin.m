#include "systray.h"
#import <Cocoa/Cocoa.h>

#if __MAC_OS_X_VERSION_MIN_REQUIRED < 101400

#ifndef NSControlStateValueOff
#define NSControlStateValueOff NSOffState
#endif

#ifndef NSControlStateValueOn
#define NSControlStateValueOn NSOnState
#endif

#endif

@interface MenuLabelView : NSView
@property(nonatomic, strong) NSString *title;
@property(nonatomic, strong) NSString *desc;
@property(nonatomic, strong) NSView *container;
@property(nonatomic, strong) NSTextField *titleLabel;
@property(nonatomic, strong) NSTextField *descLabel;
@property(nonatomic, strong) NSColor *normalColor;
@property(nonatomic, strong) NSColor *highLightColor;
@property(nonatomic, assign) id target;
@property(nonatomic, assign) SEL selector;
@property(nonatomic, strong) NSNumber *representedObject;
@end

@implementation MenuLabelView
- (id)initWithFrame:(NSRect)frame {
  self = [super initWithFrame:frame];
  if (self) {
    CGFloat paddingX = 5; // Container padding horizontally
    CGFloat paddingY = 0; // Container padding vertically
    CGFloat width = NSRectToCGRect(frame).size.width - paddingX * 2.0;
    CGFloat height = NSRectToCGRect(frame).size.height - paddingY * 2.0;
    self.container = [[NSView alloc]
        initWithFrame:NSRectFromCGRect(
                          CGRectMake(paddingX, paddingY, width, height))];
    [self addSubview:self.container];

    paddingX = 17; // Label padding horizontally
    paddingY = 2;  // Label padding vertically
    self.titleLabel = [[NSTextField alloc]
        initWithFrame:NSRectFromCGRect(CGRectMake(paddingX, paddingY,
                                                  width / 2.0,
                                                  height - paddingY * 2.0))];
    self.titleLabel.editable = NO;
    self.titleLabel.bordered = NO;
    self.titleLabel.drawsBackground = NO;
    self.titleLabel.font = [NSFont systemFontOfSize:13];
    self.titleLabel.backgroundColor = [NSColor clearColor];
    [self.container addSubview:self.titleLabel];

    self.descLabel = [[NSTextField alloc]
        initWithFrame:NSRectFromCGRect(CGRectMake(width / 2.0, paddingY,
                                                  width / 2.0 - paddingX,
                                                  height - paddingY * 2.0))];
    self.descLabel.editable = NO;
    self.descLabel.bordered = NO;
    self.descLabel.drawsBackground = NO;
    self.descLabel.font = [NSFont systemFontOfSize:13];
    self.descLabel.backgroundColor = [NSColor clearColor];
    self.descLabel.alignment = NSTextAlignmentRight;
    [self.container addSubview:self.descLabel];

    self.normalColor = self.titleLabel.textColor;
    self.highLightColor = [NSColor whiteColor];
  }
  return self;
}

- (void)setTitle:(NSString *)title {
  _title = title;
  self.titleLabel.stringValue = title;
}

- (void)setDesc:(NSString *)desc {
  _desc = desc;
  self.descLabel.stringValue = desc;
}

- (void)drawRect:(NSRect)rect {
  BOOL isHighlighted = [[self enclosingMenuItem] isHighlighted];
  if (isHighlighted) {
    self.container.layer.cornerRadius = 4;
    self.container.layer.masksToBounds = YES;
    self.container.layer.backgroundColor =
        [[[NSColor selectedContentBackgroundColor]
            colorWithAlphaComponent:0.78f] CGColor];
    [self.titleLabel setTextColor:self.highLightColor];
    [self.descLabel setTextColor:self.highLightColor];
  } else {
    [super drawRect:rect];
    self.container.layer.cornerRadius = 0;
    self.container.layer.masksToBounds = YES;
    self.container.layer.backgroundColor = [[NSColor clearColor] CGColor];
    [self.titleLabel setTextColor:self.normalColor];
    [self.descLabel setTextColor:self.normalColor];
  }
}

- (void)mouseDown:(NSEvent *)event {
  [self.target performSelectorOnMainThread:self.selector
                                withObject:self
                             waitUntilDone:YES];
}

@end

@interface MenuItem : NSObject {
@public
  NSNumber *menuId;
  NSNumber *parentMenuId;
  NSNumber *siblingId;
  NSString *title;
  short disabled;
  short checked;
}
- (id)initWithId:(int)theMenuId
    withParentMenuId:(int)theParentMenuId
       withSiblingId:(int)theSiblingId
           withTitle:(const char *)theTitle
        withDisabled:(short)theDisabled
         withChecked:(short)theChecked;
@end
@implementation MenuItem
- (id)initWithId:(int)theMenuId
    withParentMenuId:(int)theParentMenuId
       withSiblingId:(int)theSiblingId
           withTitle:(const char *)theTitle
        withDisabled:(short)theDisabled
         withChecked:(short)theChecked {
  menuId = [NSNumber numberWithInt:theMenuId];
  parentMenuId = [NSNumber numberWithInt:theParentMenuId];
  siblingId = [NSNumber numberWithInt:theSiblingId];
  title = [[NSString alloc] initWithCString:theTitle
                                   encoding:NSUTF8StringEncoding];
  disabled = theDisabled;
  checked = theChecked;
  return self;
}
@end

@interface AppDelegate : NSObject <NSApplicationDelegate>
- (void)upsertMenuItem:(MenuItem *)item;
- (IBAction)menuHandler:(id)sender;
@property(assign) IBOutlet NSWindow *window;
@end

@implementation AppDelegate {
  NSStatusItem *statusItem;
  NSMenu *menu;
  NSCondition *cond;
}

@synthesize window = _window;

- (void)applicationDidFinishLaunching:(NSNotification *)aNotification {
  self->statusItem = [[NSStatusBar systemStatusBar]
      statusItemWithLength:NSVariableStatusItemLength];
  self->menu = [[NSMenu alloc] init];
  [self->menu setAutoenablesItems:FALSE];
  [self->statusItem setMenu:self->menu];
  systray_ready();
}

- (void)applicationWillTerminate:(NSNotification *)aNotification {
  systray_on_exit();
}

- (void)setIcon:(NSImage *)image {
  statusItem.button.image = image;
  [self updateTitleButtonStyle];
}

- (void)setTitle:(NSString *)title {
  statusItem.button.title = title;
  [self updateTitleButtonStyle];
}

- (void)updateTitleButtonStyle {
  if (statusItem.button.image != nil) {
    if ([statusItem.button.title length] == 0) {
      statusItem.button.imagePosition = NSImageOnly;
    } else {
      statusItem.button.imagePosition = NSImageLeft;
    }
    statusItem.button.imageScaling = NSImageScaleProportionallyDown;
  } else {
    statusItem.button.imagePosition = NSNoImage;
  }
}

- (IBAction)menuHandler:(id)sender {
  NSNumber *menuId = [sender representedObject];
  systray_menu_item_selected(menuId.intValue);
  [self->menu cancelTracking];
}

- (void)upsertMenuItem:(MenuItem *)item {
  NSMenu *theMenu = self->menu;
  NSMenuItem *parentItem;
  if ([item->parentMenuId integerValue] > 0) {
    parentItem = find_menu_item(menu, item->parentMenuId);
    if (parentItem.hasSubmenu) {
      theMenu = parentItem.submenu;
    } else {
      theMenu = [[NSMenu alloc] init];
      [theMenu setAutoenablesItems:NO];
      [parentItem setSubmenu:theMenu];
    }
  }

  NSMenuItem *menuItem;
  MenuLabelView *labelView;
  menuItem = find_menu_item(theMenu, item->menuId);
  if (menuItem == NULL) {
    SEL selector = @selector(menuHandler:);
    if ([item->siblingId integerValue] > 0) {
      NSMenuItem *siblingItem =
          [theMenu itemWithTag:[item->siblingId integerValue]];
      NSInteger index = [theMenu indexOfItem:siblingItem];
      menuItem = [theMenu insertItemWithTitle:@""
                                       action:selector
                                keyEquivalent:@""
                                      atIndex:index];
    } else {
      menuItem = [theMenu addItemWithTitle:@""
                                    action:selector
                             keyEquivalent:@""];
    }

    if ([item->title containsString:@"\t"]) {
      MenuLabelView *labelView = [[MenuLabelView alloc]
          initWithFrame:NSRectFromCGRect(CGRectMake(0, 0, 300, 22))];
      labelView.representedObject = item->menuId;
      labelView.target = self;
      labelView.selector = selector;
      [menuItem setView:labelView];
    }

    [menuItem setRepresentedObject:item->menuId];
  }
  if ([item->title containsString:@"\t"]) {
    NSArray *listItems = [item->title componentsSeparatedByString:@"\t"];
    labelView = (MenuLabelView *)menuItem.view;
    [labelView setTitle:listItems[0]];
    [labelView setDesc:listItems[1]];
  } else {
    [menuItem setTitle:item->title];
  }

  [menuItem setTag:[item->menuId integerValue]];
  [menuItem setTarget:self];
  if (item->disabled == 1) {
    menuItem.enabled = FALSE;
  } else {
    menuItem.enabled = TRUE;
  }
  if (item->checked == 1) {
    menuItem.state = NSControlStateValueOn;
  } else {
    menuItem.state = NSControlStateValueOff;
  }
}

NSMenuItem *find_menu_item(NSMenu *ourMenu, NSNumber *menuId) {
  NSMenuItem *foundItem = [ourMenu itemWithTag:[menuId integerValue]];
  if (foundItem != NULL) {
    return foundItem;
  }
  NSArray *menu_items = ourMenu.itemArray;
  int i;
  for (i = 0; i < [menu_items count]; i++) {
    NSMenuItem *i_item = [menu_items objectAtIndex:i];
    if (i_item.hasSubmenu) {
      foundItem = find_menu_item(i_item.submenu, menuId);
      if (foundItem != NULL) {
        return foundItem;
      }
    }
  }

  return NULL;
};

- (void)addSeparator:(NSNumber *)menuId {
  NSMenuItem *menuItem = [NSMenuItem separatorItem];
  [menuItem setTag:[menuId integerValue]];
  [menu addItem:menuItem];
}

- (void)removeAllItems:(NSNumber *)menuId {
  NSMenuItem *menuItem = find_menu_item(menu, menuId);
  if (menuItem != NULL) {
    [menuItem.submenu removeAllItems];
  }
}

- (void)hideMenuItem:(NSNumber *)menuId {
  NSMenuItem *menuItem = find_menu_item(menu, menuId);
  if (menuItem != NULL) {
    [menuItem setHidden:TRUE];
  }
}

- (void)showMenuItem:(NSNumber *)menuId {
  NSMenuItem *menuItem = find_menu_item(menu, menuId);
  if (menuItem != NULL) {
    [menuItem setHidden:FALSE];
  }
}

- (void)quit {
  [NSApp terminate:self];
}

@end

void register_systray(void) {
  AppDelegate *delegate = [[AppDelegate alloc] init];
  [[NSApplication sharedApplication] setDelegate:delegate];
  // A workaround to avoid crashing on macOS versions before Catalina. Somehow
  // SIGSEGV would happen inside AppKit if [NSApp run] is called from a
  // different function, even if that function is called right after this.
  if (floor(NSAppKitVersionNumber) <= /*NSAppKitVersionNumber10_14*/ 1671) {
    [NSApp run];
  }
}

int native_loop(void) {
  if (floor(NSAppKitVersionNumber) > /*NSAppKitVersionNumber10_14*/ 1671) {
    [NSApp run];
  }
  return EXIT_SUCCESS;
}

void runInMainThread(SEL method, id object) {
  [(AppDelegate *)[NSApp delegate] performSelectorOnMainThread:method
                                                    withObject:object
                                                 waitUntilDone:YES];
}

void set_icon(const char *iconBytes, int length, bool template) {
  NSData *buffer = [NSData dataWithBytes:iconBytes length:length];
  NSImage *image = [[NSImage alloc] initWithData:buffer];
  [image setSize:NSMakeSize(19, 19)];
  image.template = template;
  runInMainThread(@selector(setIcon:), (id)image);
}

void set_title(char *ctitle) {
  NSString *title = [[NSString alloc] initWithCString:ctitle
                                             encoding:NSUTF8StringEncoding];
  free(ctitle);
  runInMainThread(@selector(setTitle:), (id)title);
}

void remove_all_items(int menuId) {
  NSNumber *mId = [NSNumber numberWithInt:menuId];
  runInMainThread(@selector(removeAllItems:), (id)mId);
}

void upsert_menu_item(int menuId, int parentMenuId, int siblingId, char *title,
                      short disabled, short checked) {
  MenuItem *item = [[MenuItem alloc] initWithId:menuId
                               withParentMenuId:parentMenuId
                                  withSiblingId:siblingId
                                      withTitle:title
                                   withDisabled:disabled
                                    withChecked:checked];
  free(title);
  runInMainThread(@selector(upsertMenuItem:), (id)item);
}

void add_separator(int menuId) {
  NSNumber *mId = [NSNumber numberWithInt:menuId];
  runInMainThread(@selector(addSeparator:), (id)mId);
}

void hide_menu_item(int menuId) {
  NSNumber *mId = [NSNumber numberWithInt:menuId];
  runInMainThread(@selector(hideMenuItem:), (id)mId);
}

void show_menu_item(int menuId) {
  NSNumber *mId = [NSNumber numberWithInt:menuId];
  runInMainThread(@selector(showMenuItem:), (id)mId);
}

void quit() { runInMainThread(@selector(quit), nil); }
