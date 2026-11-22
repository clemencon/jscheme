/* main.c -- standalone launcher: the Chez Scheme heap (petite.boot plus the
   compiled interpreter) is embedded in the executable. */
#include "scheme.h"
#include <math.h>
#include <unistd.h>

__asm__(".section .rodata\n"
        ".balign 16\n"
        ".global jscheme_boot\n"
        "jscheme_boot:\n"
        ".incbin \"jscheme.boot\"\n"
        ".global jscheme_boot_end\n"
        "jscheme_boot_end:\n"
        ".previous\n");
extern const char jscheme_boot[], jscheme_boot_end[];

static int js_isatty(int fd) { return isatty(fd); }

int main(int argc, const char *argv[]) {
  Sscheme_init(0);
  Sregister_boot_file_bytes("jscheme", (void *)jscheme_boot,
                            (iptr)(jscheme_boot_end - jscheme_boot));
  Sbuild_heap(0, 0);
  Sforeign_symbol("js_isatty", (void *)js_isatty);
  Sforeign_symbol("fmod", (void *)fmod);
  int status = Sscheme_start(argc, argv);
  Sscheme_deinit();
  return status;
}
