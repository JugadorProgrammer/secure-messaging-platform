# Сборка DLL "Кузнечик" и проверок к ней.
# Требуются g++ из MSYS2 (MinGW-w64) и mingw32-make.

CXX = g++
CXXFLAGS = -std=c++17 -Wall -Wextra -Wpedantic -Werror -O2 -Iinclude -Isrc

LIB_SOURCES = src/kuz_tables.cpp src/block_cipher.cpp src/cbc_mode.cpp src/hex.cpp src/api.cpp
LIB_HEADERS = include/kuznyechik.h src/kuz_tables.h src/block_cipher.h src/cbc_mode.h src/hex.h
IMPLIB = "-Wl,--out-implib=libkuznyechik.dll.a"
DIST = dist

all: kuznyechik.dll kuz_tests.exe kuz_cli.exe dll_check.exe

# Сама библиотека. -static убирает зависимость от libstdc++-6.dll и
# libgcc_s_seh-1.dll, иначе DLL не запустится на чужой машине.
kuznyechik.dll: $(LIB_SOURCES) $(LIB_HEADERS)
	$(CXX) $(CXXFLAGS) -DKUZ_BUILD_DLL -shared -static $(IMPLIB) $(LIB_SOURCES) -o kuznyechik.dll

# Тесты собираются статически вместе с исходниками: проверяются
# и внутренние преобразования, и C-интерфейс.
kuz_tests.exe: tests/tests.cpp $(LIB_SOURCES) $(LIB_HEADERS)
	$(CXX) $(CXXFLAGS) -DKUZ_STATIC tests/tests.cpp $(LIB_SOURCES) -o kuz_tests.exe

# Демо-клиент линкуется с DLL. Import library указывается файлом:
# с флагом -static GNU ld перестаёт искать библиотеки по -l.
kuz_cli.exe: tools/kuz_cli.cpp src/hex.cpp src/hex.h kuznyechik.dll
	$(CXX) $(CXXFLAGS) -static tools/kuz_cli.cpp src/hex.cpp libkuznyechik.dll.a -o kuz_cli.exe

# Проверка, что DLL грузится и отдаёт нужные функции (без import library).
dll_check.exe: tests/dll_check.cpp src/hex.cpp src/hex.h
	$(CXX) $(CXXFLAGS) -static tests/dll_check.cpp src/hex.cpp -o dll_check.exe

# Поставка для команды: DLL, import library, заголовок, пример и описание.
# Состав описан в docs/team-readme.md; папка dist собирается целиком.
dist: kuznyechik.dll docs/team-readme.md docs/example.cpp include/kuznyechik.h
	cmd /c "if not exist $(DIST) mkdir $(DIST) & copy /Y kuznyechik.dll $(DIST)\kuznyechik.dll & copy /Y libkuznyechik.dll.a $(DIST)\libkuznyechik.dll.a & copy /Y include\kuznyechik.h $(DIST)\kuznyechik.h & copy /Y docs\example.cpp $(DIST)\example.cpp & copy /Y docs\team-readme.md $(DIST)\README.md"

test: all
	"$(CURDIR)/kuz_tests.exe"
	"$(CURDIR)/kuz_cli.exe" selftest
	"$(CURDIR)/dll_check.exe"

clean:
	cmd /c "del /q kuznyechik.dll libkuznyechik.dll.a kuz_tests.exe kuz_cli.exe dll_check.exe 2>nul & exit 0"

.PHONY: all test clean dist
