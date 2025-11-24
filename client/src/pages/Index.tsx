import { useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CloudStatus } from "@/components/CloudStatus";
import { SearchBar } from "@/components/SearchBar";
import { VideoResult } from "@/components/VideoResult";
import { EmptyState } from "@/components/EmptyState";
import { VideoLibrary } from "@/components/VideoLibrary";
import { IndexingProgress } from "@/components/IndexingProgress";
import { VideoPlayer } from "@/components/VideoPlayer";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { Loader2, Library, Search as SearchIcon, Activity } from "lucide-react";
import { listVideos, VideoRecord } from "@/lib/api";

interface SearchResult {
  filename: string;
  timestamps: number[];
  relevanceScore?: number;
}

const Index = () => {
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const [lastQuery, setLastQuery] = useState("");
  const [selectedVideo, setSelectedVideo] = useState<{ filename: string; timestamp: number } | null>(null);

  const handleSearch = async (query: string) => {
    setLoading(true);
    setLastQuery(query);
    setHasSearched(true);

    try {
      const videos = await listVideos();
      const queryLower = query.toLowerCase();
      const matches = videos.filter((v) => v.path.toLowerCase().includes(queryLower) || v.id.toLowerCase().includes(queryLower));
      const mapped: SearchResult[] = matches.map((v) => ({
        filename: v.path,
        timestamps: [],
      }));
      setResults(mapped);
      if (mapped.length === 0) {
        toast.info("No matching videos found");
      } else {
        toast.success(`Found ${mapped.length} video(s)`);
      }
    } catch (error) {
      console.error("Search error:", error);
      toast.error("Could not connect to search service");
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenTimestamp = (filename: string, timestamp: number) => {
    setSelectedVideo({ filename, timestamp });
    toast.success(`Opening ${filename} at ${timestamp}s`);
  };

  const handleSelectVideo = (filename: string) => {
    setSelectedVideo({ filename, timestamp: 0 });
    toast.success(`Loading ${filename.split("/").pop()}`);
  };

  return (
    <div className="min-h-screen bg-background">
      <div className="container mx-auto px-6 py-8 max-w-7xl">
        {/* Header */}
        <header className="mb-8">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">
                  Semantic Video Search
                </h1>
              <p className="text-muted-foreground">
                Index and search through your videos using AI
              </p>
            </div>
          </div>

          <CloudStatus />
        </header>

        {/* Main Content */}
        <Tabs defaultValue="search" className="space-y-6">
          <TabsList className="grid w-full grid-cols-3 max-w-md">
            <TabsTrigger value="search" className="flex items-center gap-2">
              <SearchIcon className="h-4 w-4" />
              Search
            </TabsTrigger>
            <TabsTrigger value="library" className="flex items-center gap-2">
              <Library className="h-4 w-4" />
              Library
            </TabsTrigger>
            <TabsTrigger value="indexing" className="flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Indexing
            </TabsTrigger>
          </TabsList>

          {/* Search Tab */}
          <TabsContent value="search" className="space-y-6">
            <SearchBar onSearch={handleSearch} loading={loading} />

            {selectedVideo && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="text-lg font-semibold">Now Playing</h3>
                  <Badge variant="secondary" className="font-mono">
                    {selectedVideo.timestamp}s
                  </Badge>
                </div>
                <VideoPlayer 
                  startTime={selectedVideo.timestamp}
                  filename={selectedVideo.filename}
                />
              </div>
            )}

            {loading && (
              <Card className="p-12 gradient-card border-border/50">
                <div className="text-center">
                  <Loader2 className="h-12 w-12 animate-spin text-primary mx-auto mb-4" />
                  <p className="text-muted-foreground">Searching through your videos...</p>
                </div>
              </Card>
            )}

            {!loading && !hasSearched && <EmptyState type="initial" />}

            {!loading && hasSearched && results.length === 0 && (
              <EmptyState type="no-results" query={lastQuery} />
            )}

            {!loading && results.length > 0 && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h2 className="text-xl font-semibold">Search Results</h2>
                  <Badge variant="secondary">
                    {results.length} video{results.length !== 1 ? 's' : ''} found
                  </Badge>
                </div>

                <div className="space-y-3">
                  {results.map((result, idx) => (
                    <VideoResult
                      key={idx}
                      filename={result.filename}
                      timestamps={result.timestamps}
                      relevanceScore={result.relevanceScore}
                      onOpenTimestamp={handleOpenTimestamp}
                    />
                  ))}
                </div>
              </div>
            )}
          </TabsContent>

          {/* Library Tab */}
          <TabsContent value="library">
            <VideoLibrary 
              onSelectVideo={handleSelectVideo}
            />
          </TabsContent>

          {/* Indexing Tab */}
          <TabsContent value="indexing">
            <IndexingProgress />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
};

export default Index;
