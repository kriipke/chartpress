import React from "react";
import { Popover } from "../overlays/Popover.jsx";
import { GithubMark } from "./SignInButton.jsx";

/**
 * chartpress UserMenu — the authenticated user's control for the far right of the
 * top nav. An avatar trigger (GitHub avatar image, or initials fallback) opens a
 * dropdown with the user's identity and two actions: View profile and Log out
 * (log out styled as the destructive item). Fires onSelect("profile" | "logout").
 */
export function UserMenu({ name, handle, avatarUrl, onSelect, align = "end", showName = false }) {
  const [open, setOpen] = React.useState(false);
  const initials = getInitials(name || handle || "?");

  const pick = (action) => { setOpen(false); if (onSelect) onSelect(action); };

  return (
    <Popover
      open={open}
      onOpenChange={setOpen}
      align={align}
      width={232}
      trigger={<Trigger avatarUrl={avatarUrl} initials={initials} name={name} showName={showName} open={open} />}
    >
      <div style={{ margin: -14 }}>
        {/* identity header */}
        <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "13px 14px" }}>
          <Avatar avatarUrl={avatarUrl} initials={initials} size={38} />
          <div style={{ minWidth: 0 }}>
            <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text-1)", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{name || handle}</div>
            {handle && (
              <div style={{ display: "flex", alignItems: "center", gap: 5, fontSize: 12.5, color: "var(--text-muted)", marginTop: 1 }}>
                <GithubMark size={12} /> <span style={{ whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{handle}</span>
              </div>
            )}
          </div>
        </div>
        <div style={{ height: 1, background: "var(--border-subtle)" }} />
        {/* actions */}
        <div style={{ padding: 6 }}>
          <MenuItem onClick={() => pick("profile")} icon={<PersonIcon />}>View profile</MenuItem>
          <MenuItem onClick={() => pick("logout")} icon={<LogoutIcon />} danger>Log out</MenuItem>
        </div>
      </div>
    </Popover>
  );
}

function Trigger({ avatarUrl, initials, name, showName, open }) {
  const [hover, setHover] = React.useState(false);
  return (
    <span
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "inline-flex", alignItems: "center", gap: 8, padding: showName ? "4px 8px 4px 4px" : 3,
        borderRadius: 999, cursor: "pointer",
        background: open ? "var(--slate-4)" : hover ? "var(--slate-3)" : "transparent",
        transition: "background-color .15s",
      }}
    >
      <Avatar avatarUrl={avatarUrl} initials={initials} size={30} ring />
      {showName && <span style={{ fontSize: 14, fontWeight: 500, color: "var(--text-1)", fontFamily: "var(--font-sans)" }}>{name}</span>}
    </span>
  );
}

function Avatar({ avatarUrl, initials, size = 32, ring = false }) {
  const common = {
    width: size, height: size, borderRadius: "50%", flexShrink: 0,
    boxShadow: ring ? "0 0 0 1px var(--border-default)" : "none",
  };
  if (avatarUrl) {
    return <img src={avatarUrl} alt="" style={{ ...common, objectFit: "cover", display: "block" }} />;
  }
  return (
    <span style={{
      ...common, display: "inline-flex", alignItems: "center", justifyContent: "center",
      background: "var(--accent-9)", color: "#fff", fontFamily: "var(--font-sans)",
      fontSize: size * 0.42, fontWeight: 600, letterSpacing: "-0.01em",
    }}>{initials}</span>
  );
}

function MenuItem({ children, onClick, icon, danger = false }) {
  const [hover, setHover] = React.useState(false);
  const fg = danger ? "var(--red-11)" : "var(--text-1)";
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "flex", alignItems: "center", gap: 10, width: "100%", textAlign: "left",
        padding: "8px 9px", border: "none", borderRadius: "var(--radius-3)", cursor: "pointer",
        background: hover ? (danger ? "var(--red-3)" : "var(--slate-3)") : "transparent",
        color: fg, fontFamily: "var(--font-sans)", fontSize: 13.5, fontWeight: 500,
        transition: "background-color .12s",
      }}
    >
      <span style={{ display: "inline-flex", color: danger ? "var(--red-11)" : "var(--text-muted)" }}>{icon}</span>
      {children}
    </button>
  );
}

function getInitials(s) {
  const parts = String(s).replace(/^@/, "").split(/[\s._-]+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

const IconBase = (props) => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" {...props} />
);
const PersonIcon = () => <IconBase><circle cx="12" cy="8" r="4" /><path d="M4 21a8 8 0 0 1 16 0" /></IconBase>;
const LogoutIcon = () => <IconBase><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" /></IconBase>;
