import { Bars3Icon, BellIcon } from '@heroicons/react/24/outline';
import { useUIStore, useAuthStore } from '@/store';

export function Header() {
  const { setSidebarOpen } = useUIStore();
  const { tenant } = useAuthStore();

  return (
    <div className="sticky top-0 z-40 flex h-16 shrink-0 items-center gap-x-4 border-b border-gray-200 bg-white px-4 shadow-sm sm:gap-x-6 sm:px-6 lg:px-8">
      <button
        type="button"
        className="-m-2.5 p-2.5 text-gray-700 lg:hidden"
        onClick={() => setSidebarOpen(true)}
      >
        <span className="sr-only">Open sidebar</span>
        <Bars3Icon className="h-6 w-6" aria-hidden="true" />
      </button>

      {/* Separator */}
      <div className="h-6 w-px bg-gray-200 lg:hidden" aria-hidden="true" />

      <div className="flex flex-1 gap-x-4 self-stretch lg:gap-x-6">
        <div className="flex flex-1 items-center">
          {/* Breadcrumb or page title could go here */}
        </div>
        <div className="flex items-center gap-x-4 lg:gap-x-6">
          {/* Quota indicator */}
          {tenant && (
            <div className="hidden sm:flex items-center text-sm text-gray-500">
              <span className="font-medium text-gray-900">
                {tenant.monthly_quota.toLocaleString()}
              </span>
              <span className="ml-1">monthly quota</span>
            </div>
          )}

          {/* Notifications */}
          <button
            type="button"
            className="-m-2.5 p-2.5 text-gray-400 hover:text-gray-500"
          >
            <span className="sr-only">View notifications</span>
            <BellIcon className="h-6 w-6" aria-hidden="true" />
          </button>

          {/* Separator */}
          <div
            className="hidden lg:block lg:h-6 lg:w-px lg:bg-gray-200"
            aria-hidden="true"
          />

          {/* Profile dropdown */}
          <div className="flex items-center">
            <span className="hidden lg:flex lg:items-center">
              <span className="text-sm font-semibold leading-6 text-gray-900">
                {tenant?.name || 'User'}
              </span>
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

export default Header;
