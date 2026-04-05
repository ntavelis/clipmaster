//go:build linux

package clipboard

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#include <gdk-pixbuf/gdk-pixbuf.h>
#include <stdlib.h>

extern void goClipboardChanged(GtkClipboard *clipboard, GdkEvent *event, gpointer data);
extern gboolean goIdleCallback(gpointer data);

static gulong watch_clipboard(GtkClipboard *cb, gpointer data) {
    return g_signal_connect(cb, "owner-change", G_CALLBACK(goClipboardChanged), data);
}

static GtkClipboard* get_clipboard() {
    return gtk_clipboard_get(GDK_SELECTION_CLIPBOARD);
}

static void schedule_idle(gpointer data) {
    gdk_threads_add_idle(goIdleCallback, data);
}

static gboolean pixbuf_save_to_png(GdkPixbuf *pixbuf, gchar **buf, gsize *buf_size, GError **error) {
    return gdk_pixbuf_save_to_buffer(pixbuf, buf, buf_size, "png", error, NULL);
}
*/
import "C"
import (
	"context"
	"fmt"
	"sync"
	"unsafe"
)

var (
	gtkOnce    sync.Once
	gtkInitErr error

	watchMu       sync.Mutex
	watchChannels = make(map[uintptr]chan<- struct{})
	watchNextID   uintptr

	idleMu    sync.Mutex
	idleFuncs = make(map[uintptr]func())
	idleNext  uintptr
)

func initGTK() {
	gtkOnce.Do(func() {
		if C.gtk_init_check(nil, nil) == C.FALSE {
			gtkInitErr = fmt.Errorf("gtk_init_check failed")
		}
	})
}

// runOnGTKThread schedules fn to run on the GTK main thread and blocks until it completes.
func runOnGTKThread(fn func()) {
	done := make(chan struct{})

	idleMu.Lock()
	id := idleNext
	idleNext++
	idleFuncs[id] = func() {
		fn()
		close(done)
	}
	idleMu.Unlock()

	C.schedule_idle(C.gpointer(id)) //nolint:govet // gpointer is a C void* used to pass an integer ID, not a Go pointer
	<-done
}

//export goIdleCallback
func goIdleCallback(data C.gpointer) C.gboolean {
	id := uintptr(data)
	idleMu.Lock()
	fn, ok := idleFuncs[id]
	if ok {
		delete(idleFuncs, id)
	}
	idleMu.Unlock()
	if ok {
		fn()
	}
	return C.FALSE
}

//export goClipboardChanged
func goClipboardChanged(clipboard *C.GtkClipboard, event *C.GdkEvent, data C.gpointer) {
	id := uintptr(data)
	watchMu.Lock()
	ch, ok := watchChannels[id]
	watchMu.Unlock()
	if ok {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// GTKClipboard reads and writes the system clipboard using GTK3's clipboard API via CGo.
type GTKClipboard struct{}

// GetText returns the current clipboard text using GTK.
func (g GTKClipboard) GetText() (string, error) {
	initGTK()
	if gtkInitErr != nil {
		return "", gtkInitErr
	}

	var result string
	runOnGTKThread(func() {
		cb := C.get_clipboard()
		ctext := C.gtk_clipboard_wait_for_text(cb)
		if ctext != nil {
			result = C.GoString(ctext)
			C.g_free(C.gpointer(ctext))
		}
	})
	return result, nil
}

// GetImage returns PNG image bytes from the clipboard if it contains an image without text.
func (g GTKClipboard) GetImage() ([]byte, error) {
	initGTK()
	if gtkInitErr != nil {
		return nil, gtkInitErr
	}

	var result []byte
	runOnGTKThread(func() {
		cb := C.get_clipboard()

		if C.gtk_clipboard_wait_is_text_available(cb) == C.TRUE {
			return
		}
		if C.gtk_clipboard_wait_is_image_available(cb) == C.FALSE {
			return
		}

		pixbuf := C.gtk_clipboard_wait_for_image(cb)
		if pixbuf == nil {
			return
		}
		defer C.g_object_unref(C.gpointer(pixbuf))

		var buf *C.gchar
		var bufSize C.gsize
		var gerr *C.GError

		ok := C.pixbuf_save_to_png(pixbuf, &buf, &bufSize, &gerr)
		if ok == C.FALSE {
			if gerr != nil {
				C.g_error_free(gerr)
			}
			return
		}
		defer C.g_free(C.gpointer(buf))

		result = C.GoBytes(unsafe.Pointer(buf), C.int(bufSize))
	})
	return result, nil
}

// SetText writes text to the clipboard using GTK.
func (g GTKClipboard) SetText(text string) error {
	initGTK()
	if gtkInitErr != nil {
		return gtkInitErr
	}

	runOnGTKThread(func() {
		cb := C.get_clipboard()
		ctext := C.CString(text)
		defer C.free(unsafe.Pointer(ctext))
		C.gtk_clipboard_set_text(cb, ctext, C.gint(len(text)))
		C.gtk_clipboard_store(cb)
	})
	return nil
}

// SetImage writes PNG image data to the clipboard using GTK.
func (g GTKClipboard) SetImage(pngData []byte) error {
	initGTK()
	if gtkInitErr != nil {
		return gtkInitErr
	}

	var setErr error
	runOnGTKThread(func() {
		loader := C.gdk_pixbuf_loader_new()
		defer C.g_object_unref(C.gpointer(loader))

		var gerr *C.GError
		C.gdk_pixbuf_loader_write(loader, (*C.guchar)(unsafe.Pointer(&pngData[0])), C.gsize(len(pngData)), &gerr)
		if gerr != nil {
			setErr = fmt.Errorf("gdk_pixbuf_loader_write: %s", C.GoString((*C.char)(unsafe.Pointer(gerr.message))))
			C.g_error_free(gerr)
			return
		}
		C.gdk_pixbuf_loader_close(loader, nil)

		pixbuf := C.gdk_pixbuf_loader_get_pixbuf(loader)
		if pixbuf == nil {
			setErr = fmt.Errorf("gdk_pixbuf_loader_get_pixbuf returned nil")
			return
		}

		cb := C.get_clipboard()
		C.gtk_clipboard_set_image(cb, pixbuf)
		C.gtk_clipboard_store(cb)
	})
	return setErr
}

// Watch connects to GTK's owner-change signal and sends on notify when the clipboard changes.
func (g GTKClipboard) Watch(ctx context.Context, notify chan<- struct{}) error {
	initGTK()
	if gtkInitErr != nil {
		return gtkInitErr
	}

	watchMu.Lock()
	id := watchNextID
	watchNextID++
	watchChannels[id] = notify
	watchMu.Unlock()

	runOnGTKThread(func() {
		cb := C.get_clipboard()
		C.watch_clipboard(cb, C.gpointer(id)) //nolint:govet // gpointer is a C void* used to pass an integer ID, not a Go pointer
	})

	go func() {
		<-ctx.Done()
		watchMu.Lock()
		delete(watchChannels, id)
		watchMu.Unlock()
	}()

	return nil
}
