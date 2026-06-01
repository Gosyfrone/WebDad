'use client'

import { useState } from 'react'
import { BarChart2, Heart, MessageCircle, MoreHorizontal, Repeat2, Share } from 'lucide-react'

import { cn } from '@/lib/utils'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'

export interface PostCardProps {
  id: string
  name: string
  handle: string
  initials: string
  timestamp: string
  content: string
  likes: number
  comments: number
  reposts: number
  views: number
}

export function PostCard({
  name,
  handle,
  initials,
  timestamp,
  content,
  likes,
  comments,
  reposts,
  views,
}: PostCardProps) {
  const [liked, setLiked] = useState(false)
  const [likeCount, setLikeCount] = useState(likes)
  const [reposted, setReposted] = useState(false)
  const [repostCount, setRepostCount] = useState(reposts)

  function toggleLike() {
    setLiked((prev) => !prev)
    setLikeCount((prev) => (liked ? prev - 1 : prev + 1))
  }

  function toggleRepost() {
    setReposted((prev) => !prev)
    setRepostCount((prev) => (reposted ? prev - 1 : prev + 1))
  }

  return (
    <article className="flex gap-3 px-4 py-3 transition-colors hover:bg-muted/30">
      <Avatar className="mt-0.5 h-10 w-10 shrink-0">
        <AvatarFallback>{initials}</AvatarFallback>
      </Avatar>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        {/* Header */}
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-1.5 text-sm">
            <span className="truncate font-bold">{name}</span>
            <span className="shrink-0 text-muted-foreground">{handle}</span>
            <span className="shrink-0 text-muted-foreground">·</span>
            <span className="shrink-0 text-muted-foreground">{timestamp}</span>
          </div>
          <button
            aria-label="Plus d'options"
            className="shrink-0 rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-primary/10 hover:text-primary"
          >
            <MoreHorizontal className="h-4 w-4" />
          </button>
        </div>

        {/* Content */}
        <p className="whitespace-pre-wrap text-sm leading-relaxed">{content}</p>

        {/* Actions */}
        <div className="-ml-2 mt-1 flex items-center justify-between text-muted-foreground">
          <ActionButton
            icon={MessageCircle}
            count={comments}
            label="Commenter"
            className="hover:text-primary hover:bg-primary/10"
          />
          <ActionButton
            icon={Repeat2}
            count={repostCount}
            label="Reposter"
            active={reposted}
            onClick={toggleRepost}
            className="hover:text-green-500 hover:bg-green-500/10"
            activeClassName="text-green-500"
          />
          <ActionButton
            icon={Heart}
            count={likeCount}
            label="Aimer"
            active={liked}
            onClick={toggleLike}
            className="hover:text-red-500 hover:bg-red-500/10"
            activeClassName="text-red-500 fill-red-500"
          />
          <ActionButton
            icon={BarChart2}
            count={views}
            label="Vues"
            className="hover:text-primary hover:bg-primary/10"
          />
          <button
            aria-label="Partager"
            className="rounded-full p-1.5 transition-colors hover:bg-primary/10 hover:text-primary"
          >
            <Share className="h-4 w-4" />
          </button>
        </div>
      </div>
    </article>
  )
}

interface ActionButtonProps {
  icon: React.ElementType
  count: number
  label: string
  active?: boolean
  onClick?: () => void
  className?: string
  activeClassName?: string
}

function ActionButton({
  icon: Icon,
  count,
  label,
  active = false,
  onClick,
  className,
  activeClassName,
}: ActionButtonProps) {
  return (
    <button
      aria-label={label}
      onClick={onClick}
      className={cn(
        'flex items-center gap-1 rounded-full p-1.5 text-xs transition-colors',
        className,
        active && activeClassName,
      )}
    >
      <Icon className={cn('h-4 w-4', active && activeClassName)} />
      <span>{formatCount(count)}</span>
    </button>
  )
}

function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}
