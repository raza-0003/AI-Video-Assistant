import { Link } from "react-router-dom";
import { useAuth } from "../AuthContext";

export default function Navbar() {
  const { user, logout } = useAuth();

  return (
    <div className="navbar">
      <div className="container navbar-inner">
        <Link to="/" className="brand">
          <span className="rec-dot" />
          Reel
        </Link>
        {user && (
          <div className="nav-user">
            <span>{user.full_name || user.email}</span>
            <button className="btn-ghost" onClick={logout}>
              Log out
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
