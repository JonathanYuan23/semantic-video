import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Video, Clock, ExternalLink } from "lucide-react";
import { Badge } from "@/components/ui/badge";

interface VideoResultProps {
  videoId?: string;
  filename: string;
  timestamps: number[];
  relevanceScore?: number;
  rank: number;
  bestScore?: number;
  onOpenTimestamp: (filename: string, timestamp: number, videoId?: string) => void;
}

export const VideoResult = ({ 
  videoId,
  filename, 
  timestamps, 
  relevanceScore,
  rank,
  bestScore,
  onOpenTimestamp 
}: VideoResultProps) => {
  const formatTimestamp = (seconds: number) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);
    
    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${minutes}:${secs.toString().padStart(2, '0')}`;
  };

  const sim = relevanceScore ?? 0;
  const best = bestScore ?? sim;
  const relative = best > 0 ? Math.min(100, Math.round((sim / best) * 100)) : 0;
  const absolute = Math.round(sim * 100);

  return (
    <Card className="p-5 gradient-card border-border/50 hover:border-primary/30 transition-smooth group">
      <div className="flex items-start gap-4">
        <div className="flex flex-col items-center justify-center min-w-[44px]">
          <Badge variant="secondary" className="text-base font-semibold px-3 py-1">#{rank}</Badge>
          <span className="text-[10px] text-muted-foreground mt-1">rank</span>
        </div>
        
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-3 mb-3">
            <div className="flex-1 min-w-0">
              <h3 className="font-semibold text-foreground truncate group-hover:text-primary transition-smooth">
                {filename}
              </h3>
              <p className="text-sm text-muted-foreground mt-1 flex items-center gap-2">
                <Video className="h-4 w-4" />
                {timestamps.length} {timestamps.length === 1 ? 'match' : 'matches'}
              </p>
            </div>
            
            <div className="flex flex-col items-end gap-1 min-w-[120px]">
              <span className="text-xs text-muted-foreground">relative to best</span>
              <div className="flex items-center gap-2 w-full">
                <div className="flex-1 h-2 rounded-full bg-muted overflow-hidden">
                  <div
                    className="h-full bg-primary transition-[width] duration-300"
                    style={{ width: `${relative}%` }}
                  />
                </div>
                <Badge variant="secondary" className="shrink-0">
                  {relative}%
                </Badge>
              </div>
              <span className="text-[11px] text-muted-foreground">sim: {absolute}%</span>
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Clock className="h-3.5 w-3.5" />
              <span>Timestamps:</span>
            </div>
            
            <div className="flex flex-wrap gap-2">
              {timestamps.slice(0, 10).map((timestamp, idx) => (
                <Button
                  key={idx}
                  variant="outline"
                  size="sm"
                  onClick={() => onOpenTimestamp(filename, timestamp, videoId)}
                  className="font-mono text-xs border-border/50 hover:border-primary/50 hover:bg-primary/10 hover:text-primary transition-smooth"
                >
                  <Clock className="h-3 w-3 mr-1.5" />
                  {formatTimestamp(timestamp)}
                  <ExternalLink className="h-3 w-3 ml-1.5 opacity-0 group-hover:opacity-100 transition-smooth" />
                </Button>
              ))}
              
              {timestamps.length > 10 && (
                <Badge variant="secondary" className="self-center">
                  +{timestamps.length - 10} more
                </Badge>
              )}
            </div>
          </div>
        </div>
      </div>
    </Card>
  );
};
