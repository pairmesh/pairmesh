# define the name of the installer
Outfile "..\PairMesh_installer.exe"
 
# define the directory to install to, the desktop in this case as specified  
# by the predefined $DESKTOP variable
InstallDir $PROGRAMFILES\pairmesh
 
# default section
Section
 
    # define the output path for this file
    SetOutPath $INSTDIR
    
    # define what to install and place it in the output path
    File ..\PairMesh.exe

    # define uninstaller name
    WriteUninstaller $INSTDIR\uninstaller.exe

    # create a shortcut named "new shortcut" in the start menu programs directory
    # point the new shortcut at the program uninstaller
    CreateShortcut "$SMPROGRAMS\PairMesh.lnk" "$INSTDIR\PairMesh.exe"

#-------
# default section end
SectionEnd
 
# create a section to define what the uninstaller does.
# the section will always be named "Uninstall"
Section "Uninstall"

     # Remove the link from the start menu
    Delete "$SMPROGRAMS\PairMesh.lnk"

    # Delete installed file
    Delete $INSTDIR\PairMesh.exe
    
    # Delete the uninstaller
    Delete $INSTDIR\uninstaller.exe
    
    # Delete the directory
    RMDir $INSTDIR

SectionEnd