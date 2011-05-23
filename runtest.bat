@mkdir bin

8g -o bin\markup.8 src\markup.go
@if ERRORLEVEL 1 EXIT /B 1

8g -o bin\test.8 src\test.go
@if ERRORLEVEL 1 EXIT /B 1

8l -o bin\markuptest.exe bin\markup.8 bin\test.8
@if ERRORLEVEL 1 EXIT /B 1

bin\markuptest.exe
