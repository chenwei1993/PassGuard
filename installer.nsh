; PassGuard's additions to the NSIS installer.
;
; Windows caches a shortcut's icon by its path, so after an update it can
; keep drawing a blank icon that an earlier build left behind, even though
; the executable now carries one. Pointing the shortcuts at the app's icon
; explicitly, and telling the shell that they changed, makes Explorer read
; it again.
;
; The names come from mygo.config.ts and are what the generated installer
; uses: windows.displayName names the shortcuts, name names the executable.
; Keep them in sync when either changes.

!macro customInstall
  ; The install section has just made the Start menu shortcut; a desktop
  ; one exists only when the user asked for it before (or the finish page
  ; makes it after this macro).
  CreateShortCut "$SMPROGRAMS\守密.lnk" "$INSTDIR\PassGuard.exe" "" "$INSTDIR\PassGuard.exe" 0
  IfFileExists "$DESKTOP\守密.lnk" 0 +2
    CreateShortCut "$DESKTOP\守密.lnk" "$INSTDIR\PassGuard.exe" "" "$INSTDIR\PassGuard.exe" 0
  ; SHCNE_ASSOCCHANGED with SHCNF_IDLIST: have Explorer reread its icons.
  System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'
!macroend
