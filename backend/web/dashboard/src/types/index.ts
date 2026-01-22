export * from './api';

// UI-specific types
export interface NavItem {
  name: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  current?: boolean;
}

export interface TableColumn<T> {
  key: keyof T | string;
  header: string;
  render?: (item: T) => React.ReactNode;
  sortable?: boolean;
  className?: string;
}

export interface FilterOption {
  label: string;
  value: string;
}

export interface DateRange {
  start: Date;
  end: Date;
}

export interface ChartDataset {
  label: string;
  data: number[];
  borderColor?: string;
  backgroundColor?: string;
}

export interface ChartData {
  labels: string[];
  datasets: ChartDataset[];
}
