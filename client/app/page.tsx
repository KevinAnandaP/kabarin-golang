export default function Home() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold tracking-tight">Ngabarin</h1>
        <p className="text-muted-foreground">Modern Web Chat Application</p>
        <div className="flex gap-4 justify-center pt-4">
          <a 
            href="/login" 
            className="px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition"
          >
            Login
          </a>
          <a 
            href="/register" 
            className="px-6 py-2 rounded-lg border border-border hover:bg-accent transition"
          >
            Register
          </a>
        </div>
      </div>
    </div>
  );
}
