@mkdir bin

8g -o bin\markup.8 src\markup.go
@if ERRORLEVEL 1 EXIT /B 1

8g -o bin\markup-test.8 src\markup-test.go
@if ERRORLEVEL 1 EXIT /B 1

8l -o bin\markuptest.exe bin\markup.8 bin\markup-test.8
@if ERRORLEVEL 1 EXIT /B 1

bin\markuptest.exe
