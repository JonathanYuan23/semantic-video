import { useState } from "react";
import { Search, Loader2 } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

interface SearchBarProps {
  onSearch: (query: string) => void;
  loading?: boolean;
}

export const SearchBar = ({ onSearch, loading }: SearchBarProps) => {
  const [query, setQuery] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      onSearch(query.trim());
    }
  };

  return (
    <form onSubmit={handleSubmit} className="w-full">
      <div className="relative group">
        <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground transition-smooth group-focus-within:text-primary" />
        <Input
          type="text"
          placeholder="Search your videos using natural language..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="pl-12 pr-24 h-14 text-base bg-card border-border/50 focus:border-primary/50 focus:glow-primary transition-smooth"
          disabled={loading}
        />
        <Button
          type="submit"
          disabled={loading || !query.trim()}
          className="absolute right-2 top-1/2 -translate-y-1/2 gradient-primary text-primary-foreground"
        >
          {loading ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              Searching
            </>
          ) : (
            "Search"
          )}
        </Button>
      </div>
    </form>
  );
};
