import { Search, Video } from "lucide-react";

interface EmptyStateProps {
  type: "initial" | "no-results";
  query?: string;
}

export const EmptyState = ({ type, query }: EmptyStateProps) => {
  if (type === "initial") {
    return (
      <div className="flex flex-col items-center justify-center py-20 px-4 text-center">
        <div className="p-6 rounded-2xl bg-primary/10 text-primary mb-6">
          <Video className="h-12 w-12" />
        </div>
        <h2 className="text-2xl font-bold mb-3">Search Your Videos Semantically</h2>
        <p className="text-muted-foreground max-w-md">
          Enter a natural language query above to find specific moments in your indexed videos.
          The AI will search through frame embeddings to find relevant timestamps.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center justify-center py-20 px-4 text-center">
      <div className="p-6 rounded-2xl bg-muted/50 text-muted-foreground mb-6">
        <Search className="h-12 w-12" />
      </div>
      <h2 className="text-2xl font-bold mb-3">No Results Found</h2>
      <p className="text-muted-foreground max-w-md">
        Couldn't find any matches for "<span className="font-semibold text-foreground">{query}</span>".
        Try rephrasing your query or using different keywords.
      </p>
    </div>
  );
};
