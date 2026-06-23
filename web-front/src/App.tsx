import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider, useAuth } from "./auth";
import { Layout } from "./components/Layout";
import { Dashboard } from "./pages/Dashboard";
import { Login } from "./pages/Login";
import { Logs } from "./pages/Logs";
import { Settings } from "./pages/Settings";
import { Newsletters } from "./pages/Newsletters";
import { Subscribers } from "./pages/Subscribers";
import { Users } from "./pages/Users";
import type { ReactElement } from "react";

function Guard({ children }: { children: ReactElement }) {
  const { user, loading } = useAuth();
  if (loading) return <div className="spinner">loading…</div>;
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

function Routed() {
  const { user, loading } = useAuth();
  return (
    <Routes>
      <Route
        path="/login"
        element={loading ? <div className="spinner">loading…</div> : user ? <Navigate to="/" replace /> : <Login />}
      />
      <Route
        element={
          <Guard>
            <Layout />
          </Guard>
        }
      >
        <Route path="/" element={<Dashboard />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/logs" element={<Logs />} />
        <Route path="/newsletters" element={<Newsletters />} />
        <Route path="/subscribers" element={<Subscribers />} />
        <Route path="/users" element={<Users />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routed />
      </BrowserRouter>
    </AuthProvider>
  );
}
