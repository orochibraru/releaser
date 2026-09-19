#include <stdio.h>

#ifndef VERSION
#define VERSION "dev"
#endif

int main(void) {
    printf("hello from %s\n", VERSION);
    return 0;
}
