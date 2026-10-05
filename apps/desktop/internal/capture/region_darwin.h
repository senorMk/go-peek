#ifndef GOPEEK_REGION_H
#define GOPEEK_REGION_H
typedef struct { int state; double x, y, width, height; } GoPeekRegion;
int GoPeekBeginRegion(void);
GoPeekRegion GoPeekPollRegion(void);
void GoPeekCancelRegion(void);
#endif
