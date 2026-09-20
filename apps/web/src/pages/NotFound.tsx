import { Link } from "react-router-dom";

const NotFound = () => (
  <div className="flex min-h-screen items-center justify-center bg-background">
    <div className="text-center">
      <h1 className="mb-4 text-4xl font-bold">404</h1>
      <p className="mb-4 text-xl text-muted-foreground">Page not found</p>
      <Link to="/files" className="text-primary underline">
        Back to uploads
      </Link>
    </div>
  </div>
);

export default NotFound;
