import Icon from '../util/Icon'
import { Fragment } from 'react'
import styles from '../../styles/layout/Navbar.module.css'
import Button from '../util/Button'
const Navbar = ({ auth, handleAction }) => {
	return (
		<Fragment>
			<nav className={styles.NavBar}>
				{auth ? (
					<Fragment>
						<div onClick={() => handleAction('logout')}>
							<Icon name="logout" /> Logout
						</div>
					</Fragment>
				) : (
					<Fragment>
						<Button color="black" bgColor="transparent" action={() => handleAction('login')}>Login</Button>
						<Button color="white" bgColor="#6868AC" action={() => handleAction('signup')}>Signup</Button>
					</Fragment>
				)}
			</nav>
		</Fragment>
	)
}

export default Navbar
