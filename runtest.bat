@mkdir bin

8g -o bin\markup.8 html.go markup.go
@if ERRORLEVEL 1 EXIT /B 1

8g -o bin\test.8 -I bin test.go 
@if ERRORLEVEL 1 EXIT /B 1

8l -o bin\markuptest.exe -L bin bin\test.8
@if ERRORLEVEL 1 EXIT /B 1

bin\markuptest.exe
