"use client";

import { useState, useRef, useEffect } from "react";
import { IconPencil, IconCheck, IconX } from "@tabler/icons-react";

interface InlineEditProps {
  value: string;
  placeholder?: string;
  onSave: (value: string) => Promise<boolean>;
  multiline?: boolean;
  className?: string;
  editClassName?: string;
}

export function InlineEdit({
  value,
  placeholder = "Click to edit...",
  onSave,
  multiline = false,
  className = "",
  editClassName = "",
}: InlineEditProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const [saving, setSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement>(null);

  useEffect(() => {
    setDraft(value);
  }, [value]);

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [editing]);

  const handleSave = async () => {
    if (draft === value) {
      setEditing(false);
      return;
    }
    setSaving(true);
    const ok = await onSave(draft);
    setSaving(false);
    if (ok) {
      setEditing(false);
    }
  };

  const handleCancel = () => {
    setDraft(value);
    setEditing(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !multiline) {
      e.preventDefault();
      handleSave();
    }
    if (e.key === "Enter" && multiline && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSave();
    }
    if (e.key === "Escape") {
      handleCancel();
    }
  };

  if (editing) {
    return (
      <div className={`flex items-start gap-2 ${editClassName}`}>
        {multiline ? (
          <textarea
            ref={inputRef as React.RefObject<HTMLTextAreaElement>}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={handleKeyDown}
            rows={3}
            className="flex-1 bg-white/[0.03] border border-white/[0.12] rounded-lg px-3 py-2 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.2] transition-colors resize-none"
            placeholder={placeholder}
          />
        ) : (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={handleKeyDown}
            className="flex-1 bg-white/[0.03] border border-white/[0.12] rounded-lg px-3 py-2 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.2] transition-colors"
            placeholder={placeholder}
          />
        )}
        <button
          onClick={handleSave}
          disabled={saving}
          className="p-1.5 rounded-lg text-green-400/70 hover:text-green-400 hover:bg-green-500/10 transition-colors disabled:opacity-30"
        >
          {saving ? (
            <div className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
          ) : (
            <IconCheck size={14} />
          )}
        </button>
        <button
          onClick={handleCancel}
          className="p-1.5 rounded-lg text-muted-foreground/40 hover:text-foreground/60 hover:bg-white/[0.04] transition-colors"
        >
          <IconX size={14} />
        </button>
      </div>
    );
  }

  return (
    <div
      onClick={() => setEditing(true)}
      className={`group cursor-pointer relative ${className}`}
    >
      <span className={value ? "" : "text-muted-foreground/30 italic"}>
        {value || placeholder}
      </span>
      <IconPencil
        size={12}
        className="inline-block ml-2 opacity-0 group-hover:opacity-40 transition-opacity"
      />
    </div>
  );
}

interface InlineTagsEditProps {
  tags: string[];
  onSave: (tags: string[]) => Promise<boolean>;
  color?: string;
}

export function InlineTagsEdit({
  tags,
  onSave,
  color = "bg-purple-500/10 text-purple-300/80 border-purple-500/20",
}: InlineTagsEditProps) {
  const [editing, setEditing] = useState(false);
  const [newTag, setNewTag] = useState("");
  const [saving, setSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus();
    }
  }, [editing]);

  const handleAdd = async () => {
    const tag = newTag.trim();
    if (!tag || tags.includes(tag)) {
      setNewTag("");
      return;
    }
    setSaving(true);
    const ok = await onSave([...tags, tag]);
    setSaving(false);
    if (ok) {
      setNewTag("");
    }
  };

  const handleRemove = async (tag: string) => {
    setSaving(true);
    await onSave(tags.filter((t) => t !== tag));
    setSaving(false);
  };

  return (
    <div className="flex flex-wrap gap-2 items-center">
      {tags.map((tag) => (
        <span
          key={tag}
          className={`px-2.5 py-1 rounded-full text-[11px] font-mono border ${color} ${editing ? "pr-1.5" : ""}`}
        >
          {tag}
          {editing && (
            <button
              onClick={() => handleRemove(tag)}
              className="ml-1.5 text-red-400/60 hover:text-red-400 transition-colors"
              disabled={saving}
            >
              <IconX size={10} />
            </button>
          )}
        </span>
      ))}
      {editing ? (
        <div className="flex items-center gap-1">
          <input
            ref={inputRef}
            value={newTag}
            onChange={(e) => setNewTag(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                handleAdd();
              }
              if (e.key === "Escape") setEditing(false);
            }}
            placeholder="new trait..."
            className="w-24 bg-white/[0.03] border border-white/[0.12] rounded-full px-2.5 py-1 text-[11px] font-mono text-foreground/70 focus:outline-none focus:border-white/[0.2] transition-colors"
          />
          <button
            onClick={() => setEditing(false)}
            className="p-1 text-muted-foreground/40 hover:text-foreground/60 transition-colors"
          >
            <IconCheck size={12} />
          </button>
        </div>
      ) : (
        <button
          onClick={() => setEditing(true)}
          className="px-2.5 py-1 rounded-full text-[11px] font-mono border border-dashed border-white/[0.1] text-muted-foreground/30 hover:text-muted-foreground/50 hover:border-white/[0.2] transition-colors"
        >
          + add
        </button>
      )}
    </div>
  );
}
