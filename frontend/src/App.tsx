import { NavLink, Route, Routes } from 'react-router-dom'

type IconName = 'home' | 'book' | 'check' | 'users' | 'settings'

type NavigationItem = {
  label: string
  path: string
  icon: IconName
}

const navigationItems: NavigationItem[] = [
  { label: 'Overview', path: '/', icon: 'home' },
  { label: 'My classes', path: '/classes', icon: 'book' },
  { label: 'Assignments', path: '/assignments', icon: 'check' },
  { label: 'Students', path: '/students', icon: 'users' },
]

function Icon({ name }: { name: IconName }) {
  const paths: Record<IconName, string> = {
    home: 'M3 10.5 12 3l9 7.5M5.5 9v10h13V9M9 19v-5h6v5',
    book: 'M5 4.5h11.5A2.5 2.5 0 0 1 19 7v12.5H7.5A2.5 2.5 0 0 1 5 17V4.5ZM5 17a2.5 2.5 0 0 1 2.5-2.5H19',
    check: 'm5 12 4 4L19 6',
    users: 'M16 20v-1.5a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4V20M9.5 10.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7ZM17 11a3 3 0 1 0 0-6M17 14.5a4 4 0 0 1 4 4V20',
    settings: 'M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7ZM19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-1.8 1.8-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.6v.1h-2.6V20a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1-1.8-1.8.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.6-1H6v-2.6h.1a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9l-.1-.1 1.8-1.8.1.1a1.7 1.7 0 0 0 1.9.3 1.7 1.7 0 0 0 1-1.6V5h2.6v.1a1.7 1.7 0 0 0 1 1.6 1.7 1.7 0 0 0 1.9-.3l.1-.1 1.8 1.8-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.6 1h.1v2.6h-.1a1.7 1.7 0 0 0-1.6 1Z',
  }

  return (
    <svg aria-hidden="true" className="icon" viewBox="0 0 24 24" fill="none">
      <path d={paths[name]} stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.8" />
    </svg>
  )
}

function Sidebar() {
  return (
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-mark" aria-hidden="true">M</span>
        <span>MiniClass</span>
      </div>

      <nav className="primary-nav" aria-label="Primary navigation">
        <p className="nav-label">Workspace</p>
        {navigationItems.map((item) => (
          <NavLink
            className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
            end={item.path === '/'}
            key={item.path}
            to={item.path}
          >
            <Icon name={item.icon} />
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="sidebar-footer">
        <NavLink className="nav-item" to="/settings">
          <Icon name="settings" />
          <span>Settings</span>
        </NavLink>
        <div className="profile-card">
          <div className="avatar" aria-hidden="true">CM</div>
          <div>
            <strong>Christopher Mott</strong>
            <span>Teacher account</span>
          </div>
          <span className="more-button" aria-hidden="true">•••</span>
        </div>
      </div>
    </aside>
  )
}

function Header() {
  return (
    <header className="topbar">
      <div>
        <p className="eyebrow">Tuesday, August 22, 2026</p>
        <h1>Good morning, Christopher</h1>
      </div>
      <div className="topbar-actions">
        <button className="icon-button" type="button" aria-label="View notifications">
          <span className="notification-dot" />
          <svg aria-hidden="true" className="icon" viewBox="0 0 24 24" fill="none">
            <path d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9ZM10 21h4" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.8" />
          </svg>
        </button>
        <button className="help-button" type="button">?</button>
      </div>
    </header>
  )
}

function StatCard({ label, value, detail, tone }: { label: string; value: string; detail: string; tone: string }) {
  return (
    <article className="stat-card">
      <div className={`stat-icon ${tone}`} aria-hidden="true"><span /></div>
      <div>
        <p>{label}</p>
        <strong>{value}</strong>
        <span className="stat-detail">{detail}</span>
      </div>
    </article>
  )
}

function Overview() {
  return (
    <div className="page-content">
      <section className="welcome-panel">
        <div>
          <p className="eyebrow accent">Your teaching hub</p>
          <h2>Everything for your classroom, in one place.</h2>
          <p className="welcome-copy">Plan lessons, keep students on track, and make every class count.</p>
        </div>
        <div className="welcome-art" aria-hidden="true">
          <span className="sun" />
          <span className="cloud cloud-one" />
          <span className="cloud cloud-two" />
          <span className="hill hill-one" />
          <span className="hill hill-two" />
        </div>
      </section>

      <section className="section-block" aria-labelledby="snapshot-heading">
        <div className="section-heading">
          <div>
            <p className="eyebrow">At a glance</p>
            <h2 id="snapshot-heading">Your classroom snapshot</h2>
          </div>
          <button className="text-button" type="button">View reports <span aria-hidden="true">→</span></button>
        </div>
        <div className="stats-grid">
          <StatCard detail="Ready to teach" label="Active classes" tone="blue" value="4" />
          <StatCard detail="Across all classes" label="Total students" tone="orange" value="86" />
          <StatCard detail="Due this week" label="Assignments" tone="purple" value="12" />
          <StatCard detail="This semester" label="Completion rate" tone="green" value="91%" />
        </div>
      </section>

      <section className="section-block lower-grid" aria-label="Upcoming and quick actions">
        <div className="card upcoming-card">
          <div className="card-heading">
            <div>
              <p className="eyebrow">Keep an eye on</p>
              <h2>Upcoming assignments</h2>
            </div>
            <NavLink className="text-button" to="/assignments">See all <span aria-hidden="true">→</span></NavLink>
          </div>
          <div className="empty-state">
            <div className="empty-icon" aria-hidden="true"><Icon name="check" /></div>
            <strong>Your assignment list is ready</strong>
            <p>New assignments and due dates will appear here.</p>
            <NavLink className="secondary-button" to="/assignments">Explore assignments</NavLink>
          </div>
        </div>

        <div className="card quick-card">
          <div className="card-heading">
            <div>
              <p className="eyebrow">Jump right in</p>
              <h2>Quick actions</h2>
            </div>
          </div>
          <div className="quick-actions">
            <NavLink to="/classes"><span className="quick-icon blue"><Icon name="book" /></span>Manage classes <span aria-hidden="true">→</span></NavLink>
            <NavLink to="/students"><span className="quick-icon orange"><Icon name="users" /></span>View students <span aria-hidden="true">→</span></NavLink>
            <NavLink to="/assignments"><span className="quick-icon purple"><Icon name="check" /></span>Create assignment <span aria-hidden="true">→</span></NavLink>
          </div>
        </div>
      </section>
    </div>
  )
}

function PlaceholderPage({ title }: { title: string }) {
  return (
    <div className="page-content placeholder-page">
      <p className="eyebrow accent">MiniClass workspace</p>
      <h2>{title}</h2>
      <p>This area is ready for the next feature to be added.</p>
      <NavLink className="secondary-button" to="/">Back to overview</NavLink>
    </div>
  )
}

function App() {
  return (
    <div className="app-shell">
      <Sidebar />
      <main className="main-content">
        <Header />
        <Routes>
          <Route path="/" element={<Overview />} />
          <Route path="/classes" element={<PlaceholderPage title="My classes" />} />
          <Route path="/assignments" element={<PlaceholderPage title="Assignments" />} />
          <Route path="/students" element={<PlaceholderPage title="Students" />} />
          <Route path="/settings" element={<PlaceholderPage title="Settings" />} />
          <Route path="*" element={<PlaceholderPage title="Page not found" />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
