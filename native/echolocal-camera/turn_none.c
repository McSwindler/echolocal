#include <stddef.h>

#include "turn.h"

turner *turn_open(ANativeWindow *main, int mw, int mh, ANativeWindow *sub, int sw, int sh, int cw, int ch, int quarters,
	ANativeWindow **camera) {
	return NULL;
}

int turn_frame(turner *t, unsigned char **still) { return 0; }

void turn_close(turner *t) {}
