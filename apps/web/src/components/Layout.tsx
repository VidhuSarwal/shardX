import { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { Cloud, Files, User, LogOut } from 'lucide-react';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { getAuthEmail } from '@/lib/api';
import { getInitialsFromName } from '@/lib/utils';

interface LayoutProps {
  children: ReactNode;
}

export const Layout = ({ children }: LayoutProps) => {
  const { isAuthenticated, logout } = useAuth();
  const location = useLocation();

  // The backend JWT only carries the user's ObjectID, so the email is remembered at login.
  const accountEmail = getAuthEmail();

  const isActive = (path: string) => location.pathname === path;

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border bg-card/50 backdrop-blur-sm sticky top-0 z-50">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <Link to="/files" className="flex items-center gap-2 text-xl font-bold">
              <Cloud className="w-6 h-6 text-primary" />
              <span className="bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
                shardX
              </span>
            </Link>

            {isAuthenticated ? (
              <nav className="flex items-center gap-2">
                <Button
                  variant={isActive('/files') ? 'default' : 'ghost'}
                  asChild
                  size="sm"
                >
                  <Link to="/files">
                    <Files className="w-4 h-4 mr-2" />
                    Files
                  </Link>
                </Button>

                <Button
                  variant={isActive('/profile') ? 'default' : 'ghost'}
                  asChild
                  size="sm"
                >
                  <Link to="/profile">
                    <User className="w-4 h-4 mr-2" />
                    Profile
                  </Link>
                </Button>

                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="sm" className="p-1.5">
                      <Avatar className="h-8 w-8">
                        <AvatarFallback className="text-xs">
                          {getInitialsFromName(accountEmail ?? 'Account')}
                        </AvatarFallback>
                      </Avatar>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-56">
                    <DropdownMenuLabel className="font-normal">
                      <div className="flex flex-col">
                        <span className="text-sm font-medium leading-none">{accountEmail ?? 'Account'}</span>
                        <span className="text-xs text-muted-foreground">shardX user</span>
                      </div>
                    </DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem asChild>
                      <Link to="/profile">Profile</Link>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={logout}>
                      <LogOut className="w-4 h-4 mr-2" /> Logout
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </nav>
            ) : (
              <div />
            )}
          </div>
        </div>
      </header>

      <main className="container mx-auto px-4 py-8">
        {children}
      </main>
    </div>
  );
};
