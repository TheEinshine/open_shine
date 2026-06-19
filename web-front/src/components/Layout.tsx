import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../auth";

export function Layout() {
  const { user, logout } = useAuth();
  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <span className="dot" />
          <span>Open Shine</span>
        </div>
        <NavLink to="/" end className="nav-link">
          <span>Dashboard</span>
        </NavLink>
        <NavLink to="/settings" className="nav-link">
          <span>Settings</span>
        </NavLink>
        <NavLink to="/logs" className="nav-link">
          <span>Logs</span>
        </NavLink>
        <NavLink to="/newsletters" className="nav-link">
          <span>Newsletters</span>
        </NavLink>
        <NavLink to="/subscribers" className="nav-link">
          <span>Subscribers</span>
        </NavLink>
        <div className="spacer" />
        <div className="who">{user?.email}</div>
        <button className="ghost" onClick={() => logout()}>
          Sign out
        </button>
      </aside>
      <main className="main">
        <Outlet />
      </main>
    </div>
  );
}
