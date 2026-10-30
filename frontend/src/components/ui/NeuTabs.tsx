import React from 'react';

export interface TabOption<T extends string = string> {
  id: T;
  label: string;
  count?: number;
  icon?: React.ReactNode;
}

export interface NeuTabsProps<T extends string = string> {
  tabs: TabOption<T>[];
  activeTab: T;
  onChange: (tabId: T) => void;
  className?: string;
}

export function NeuTabs<T extends string = string>({
  tabs,
  activeTab,
  onChange,
  className = '',
}: NeuTabsProps<T>) {
  return (
    <div
      className={`inline-flex items-center p-1.5 neu-container-inset border border-neu-border/50 max-w-full overflow-x-auto ${className}`}
    >
      {tabs.map((tab) => {
        const isActive = activeTab === tab.id;
        return (
          <button
            key={tab.id}
            type="button"
            onClick={() => onChange(tab.id)}
            className={`flex items-center gap-2 px-4 py-2 rounded-neu-sm text-xs font-semibold whitespace-nowrap transition-all duration-200 cursor-pointer ${
              isActive
                ? 'bg-neu-bg text-forest shadow-neu-raised-sm border border-neu-border/60'
                : 'text-slateneu hover:text-charcoal hover:bg-black/[0.02]'
            }`}
          >
            {tab.icon && <span className="inline-flex shrink-0">{tab.icon}</span>}
            <span>{tab.label}</span>
            {typeof tab.count === 'number' && (
              <span
                className={`px-1.5 py-0.5 rounded-full text-[10px] font-bold ${
                  isActive
                    ? 'bg-forest/15 text-forest'
                    : 'bg-slateneu/15 text-slateneu'
                }`}
              >
                {tab.count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
