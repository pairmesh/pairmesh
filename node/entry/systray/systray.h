#include "stdbool.h"

extern void systray_ready();
extern void systray_on_exit();
extern void systray_menu_item_selected(int menu_id);
void register_systray(void);
int native_loop(void);

void set_icon(const char* iconBytes, int length, bool template);
void set_title(char* title);
void remove_all_items(int menuId);
void upsert_menu_item(int menuId, int parentMenuId, int siblingId, char* title, short disabled, short checked);
void add_separator(int menuId);
void hide_menu_item(int menuId);
void show_menu_item(int menuId);
void quit();
