Set WshShell = CreateObject("WScript.Shell")
WshShell.CurrentDirectory = "D:\software\fp"
WshShell.Run """D:\software\fp\mfs100-auth-service.exe""", 0, False
