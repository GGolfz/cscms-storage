import Icon from '../util/Icon'
import { Fragment } from 'react'
import styles from '../../styles/layout/Sidebar.module.css'
const Sidebar = ({ currentRoute = 'file', handleChangeRoute }) => {
	const routes = ['file', 'image']
	const activeColor = '#6F6CBB'
	const inactiveColor = '#B0B2E2'
	return (
		<Fragment>
			<div className={styles.SideBar}>
				{routes.map(route => (
					<div
						className={`${styles.SideBar__item} ${
							currentRoute === route ? styles.active : ''
						}`}
						key={route}
						onClick={() => handleChangeRoute(route)}
					>
						<Icon
							name={route}
							color={currentRoute === route ? activeColor : inactiveColor}
						/>
					</div>
				))}
			</div>
		</Fragment>
	)
}

export default Sidebar
