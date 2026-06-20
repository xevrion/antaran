package tray

// SNI (StatusNotifierItem) tray icon over DBus.
// Pure Go — no GTK, no C. Wails already owns the GTK main loop;
// this registers an SNI item on the session bus so the compositor
// (KDE, Hyprland+waybar, etc.) picks it up independently.

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const (
	sniInterface = "org.kde.StatusNotifierItem"
	sniPath      = dbus.ObjectPath("/StatusNotifierItem")
	sniWatcherService = "org.kde.StatusNotifierWatcher"
	sniWatcherPath    = dbus.ObjectPath("/StatusNotifierWatcher")
)

// SNIItem represents our tray icon on the session bus.
type SNIItem struct {
	conn       *dbus.Conn
	service    string
	tooltip    string
	title      string
	onActivate func() // called on left-click
}

// NewSNIItem registers a StatusNotifierItem on the session bus.
// onActivate is called (in a new goroutine) when the user clicks the icon.
func NewSNIItem(onActivate func()) (*SNIItem, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("dbus session connect: %w", err)
	}

	svcName := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(svcName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		conn.Close()
		return nil, fmt.Errorf("dbus name %q unavailable (reply=%v, err=%v)", svcName, reply, err)
	}

	item := &SNIItem{
		conn:       conn,
		service:    svcName,
		tooltip:    "antaran",
		title:      "antaran",
		onActivate: onActivate,
	}

	conn.Export(item, sniPath, sniInterface)
	conn.Export(introspect.Introspectable(sniIntrospectXML), sniPath,
		"org.freedesktop.DBus.Introspectable")

	// Tell the watcher we exist
	watcher := conn.Object(sniWatcherService, sniWatcherPath)
	if call := watcher.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem",
		0, svcName); call.Err != nil {
		fmt.Fprintf(os.Stderr, "antaran-tray: SNI watcher register: %v\n", call.Err)
	}

	return item, nil
}

// SetTooltip updates the tooltip text and emits a NewToolTip signal.
func (s *SNIItem) SetTooltip(text string) {
	s.tooltip = text
	_ = s.conn.Emit(sniPath, sniInterface+".NewToolTip")
}

// SetTitle updates the icon label/title and emits a NewTitle signal.
func (s *SNIItem) SetTitle(text string) {
	s.title = text
	_ = s.conn.Emit(sniPath, sniInterface+".NewTitle")
}

// Close releases the DBus name.
func (s *SNIItem) Close() {
	_, _ = s.conn.ReleaseName(s.service)
	s.conn.Close()
}

// --- DBus method/property handlers ---

func (s *SNIItem) GetId() (string, *dbus.Error)     { return "antaran", nil }
func (s *SNIItem) GetTitle() (string, *dbus.Error)  { return s.title, nil }
func (s *SNIItem) GetStatus() (string, *dbus.Error) { return "Active", nil }
func (s *SNIItem) GetIconName() (string, *dbus.Error) {
	return "application-x-executable", nil
}
func (s *SNIItem) GetIconPixmap() ([]struct {
	W, H int32
	Data []byte
}, *dbus.Error) {
	return nil, nil
}
func (s *SNIItem) GetMenu() (dbus.ObjectPath, *dbus.Error) {
	return dbus.ObjectPath("/NO_DBUSMENU"), nil
}
func (s *SNIItem) GetItemIsMenu() (bool, *dbus.Error) { return false, nil }

func (s *SNIItem) GetToolTip() (struct {
	IconName   string
	IconPixmap []struct {
		W, H int32
		Data []byte
	}
	Title string
	Body  string
}, *dbus.Error) {
	type tt struct {
		IconName   string
		IconPixmap []struct {
			W, H int32
			Data []byte
		}
		Title string
		Body  string
	}
	return tt{Title: s.tooltip}, nil
}

// Activate is called by the compositor on left-click.
func (s *SNIItem) Activate(x, y int32) *dbus.Error {
	if s.onActivate != nil {
		go s.onActivate()
	}
	return nil
}

func (s *SNIItem) SecondaryActivate(x, y int32) *dbus.Error { return nil }
func (s *SNIItem) Scroll(delta int32, orientation string) *dbus.Error { return nil }
func (s *SNIItem) ContextMenu(x, y int32) *dbus.Error { return nil }

const sniIntrospectXML = `
<node>
  <interface name="org.kde.StatusNotifierItem">
    <method name="Activate">
      <arg name="x" type="i" direction="in"/>
      <arg name="y" type="i" direction="in"/>
    </method>
    <method name="SecondaryActivate">
      <arg name="x" type="i" direction="in"/>
      <arg name="y" type="i" direction="in"/>
    </method>
    <method name="ContextMenu">
      <arg name="x" type="i" direction="in"/>
      <arg name="y" type="i" direction="in"/>
    </method>
    <method name="Scroll">
      <arg name="delta" type="i" direction="in"/>
      <arg name="orientation" type="s" direction="in"/>
    </method>
    <property name="Id"          type="s" access="read"/>
    <property name="Title"       type="s" access="read"/>
    <property name="Status"      type="s" access="read"/>
    <property name="IconName"    type="s" access="read"/>
    <property name="ToolTip"     type="(sa(iiay)ss)" access="read"/>
    <property name="Menu"        type="o" access="read"/>
    <property name="ItemIsMenu"  type="b" access="read"/>
    <signal name="NewTitle"/>
    <signal name="NewToolTip"/>
    <signal name="NewIcon"/>
    <signal name="NewStatus"><arg type="s"/></signal>
  </interface>
</node>`
