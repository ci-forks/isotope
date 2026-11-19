/*
 * Make a desktop OpenGL context through EGL with libepoxy and call a GL
 * function, the way Xwayland's glamor and GTK do. libepoxy loads the GL entry
 * points of a desktop context from libOpenGL.so.0 or libGL.so.1, and aborts
 * the process when it finds neither ("Couldn't open libGL.so.1 or
 * libOpenGL.so.0"). Run with EGL_PLATFORM=surfaceless and
 * LIBGL_ALWAYS_SOFTWARE=1 so llvmpipe answers without a GPU or a display.
 *
 * libepoxy is dlopened, not linked, so the probe needs only libc to build and
 * runs against the libepoxy of the image under test. libepoxy exports every
 * EGL and GL function as a pointer named epoxy_<name>.
 *
 * With the argument "gles" it makes an OpenGL ES 2 context instead, which is
 * what the compositors (wlroots, cosmic-comp, niri) render with.
 *
 * Exit status: 0 when GL answered, 1 on an EGL error, and libepoxy's abort
 * (SIGABRT) when no desktop GL library is found.
 */
#include <dlfcn.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef void *EGLDisplay;
typedef void *EGLConfig;
typedef void *EGLContext;
typedef void *EGLSurface;
typedef int32_t EGLint;
typedef unsigned int EGLenum;
typedef unsigned int EGLBoolean;

#define EGL_NONE 0x3038
#define EGL_RENDERABLE_TYPE 0x3040
#define EGL_SURFACE_TYPE 0x3033
#define EGL_OPENGL_BIT 0x0008
#define EGL_OPENGL_API 0x30A2
#define EGL_OPENGL_ES_API 0x30A0
#define EGL_OPENGL_ES2_BIT 0x0004
#define EGL_CONTEXT_CLIENT_VERSION 0x3098
#define EGL_PLATFORM_SURFACELESS_MESA 0x31DD
#define GL_VENDOR 0x1F00
#define GL_RENDERER 0x1F01
#define GL_VERSION 0x1F02

typedef void *(*GetProcAddressFn)(const char *);
typedef EGLDisplay (*GetPlatformDisplayFn)(EGLenum, void *, const EGLint *);
typedef EGLBoolean (*InitializeFn)(EGLDisplay, EGLint *, EGLint *);
typedef EGLBoolean (*BindAPIFn)(EGLenum);
typedef EGLBoolean (*ChooseConfigFn)(EGLDisplay, const EGLint *, EGLConfig *, EGLint, EGLint *);
typedef EGLContext (*CreateContextFn)(EGLDisplay, EGLConfig, EGLContext, const EGLint *);
typedef EGLBoolean (*MakeCurrentFn)(EGLDisplay, EGLSurface, EGLSurface, EGLContext);
typedef const unsigned char *(*GetStringFn)(unsigned int);
typedef int (*IsDesktopGLFn)(void);

static void *handle;

static void *sym(const char *name)
{
	void *p = dlsym(handle, name);
	if (!p) {
		fprintf(stderr, "gl-probe: libepoxy has no %s\n", name);
		exit(1);
	}
	return p;
}

/* The function behind libepoxy's pointer epoxy_<name>. */
#define EPOXY(type, name) (*(type *)sym("epoxy_" #name))

int main(int argc, char **argv)
{
	int gles = argc > 1 && strcmp(argv[1], "gles") == 0;

	handle = dlopen("libepoxy.so.0", RTLD_NOW);
	if (!handle) {
		fprintf(stderr, "gl-probe: %s\n", dlerror());
		return 1;
	}

	GetProcAddressFn getProcAddress = EPOXY(GetProcAddressFn, eglGetProcAddress);
	GetPlatformDisplayFn getPlatformDisplay =
		(GetPlatformDisplayFn)getProcAddress("eglGetPlatformDisplayEXT");
	if (!getPlatformDisplay) {
		fprintf(stderr, "gl-probe: no eglGetPlatformDisplayEXT\n");
		return 1;
	}
	EGLDisplay dpy = getPlatformDisplay(EGL_PLATFORM_SURFACELESS_MESA, NULL, NULL);
	EGLint major = 0, minor = 0;
	if (!dpy || !EPOXY(InitializeFn, eglInitialize)(dpy, &major, &minor)) {
		fprintf(stderr, "gl-probe: eglInitialize failed on the surfaceless platform\n");
		return 1;
	}
	if (!EPOXY(BindAPIFn, eglBindAPI)(gles ? EGL_OPENGL_ES_API : EGL_OPENGL_API)) {
		fprintf(stderr, "gl-probe: eglBindAPI failed\n");
		return 1;
	}

	const EGLint cfg_attrs[] = {
		EGL_RENDERABLE_TYPE, gles ? EGL_OPENGL_ES2_BIT : EGL_OPENGL_BIT,
		EGL_SURFACE_TYPE, 0, EGL_NONE
	};
	EGLConfig cfg = NULL;
	EGLint n = 0;
	EPOXY(ChooseConfigFn, eglChooseConfig)(dpy, cfg_attrs, &cfg, 1, &n);
	/* n == 0 leaves cfg as EGL_NO_CONFIG_KHR, which Mesa accepts. */

	const EGLint es_attrs[] = { EGL_CONTEXT_CLIENT_VERSION, 2, EGL_NONE };
	const EGLint gl_attrs[] = { EGL_NONE };
	EGLContext ctx = EPOXY(CreateContextFn, eglCreateContext)(dpy, cfg, NULL, gles ? es_attrs : gl_attrs);
	if (!ctx) {
		fprintf(stderr, "gl-probe: eglCreateContext failed\n");
		return 1;
	}
	if (!EPOXY(MakeCurrentFn, eglMakeCurrent)(dpy, NULL, NULL, ctx)) {
		fprintf(stderr, "gl-probe: eglMakeCurrent failed\n");
		return 1;
	}

	/* The first GL call: libepoxy resolves it from libOpenGL.so.0 or libGL.so.1. */
	GetStringFn getString = EPOXY(GetStringFn, glGetString);
	const unsigned char *version = getString(GL_VERSION);
	const unsigned char *renderer = getString(GL_RENDERER);
	if (!version || !renderer) {
		fprintf(stderr, "gl-probe: glGetString returned NULL\n");
		return 1;
	}
	int desktop = ((IsDesktopGLFn)sym("epoxy_is_desktop_gl"))();
	printf("EGL %d.%d\nGL_VENDOR: %s\nGL_RENDERER: %s\nGL_VERSION: %s\ndesktop GL: %s\n",
	       major, minor, getString(GL_VENDOR), renderer, version, desktop ? "yes" : "no");
	return desktop == !gles ? 0 : 1;
}
