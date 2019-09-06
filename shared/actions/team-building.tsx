import logger from '../logger'
import * as Constants from '../constants/team-building'
import * as TeamBuildingTypes from '../constants/types/team-building'
import * as TeamBuildingGen from './team-building-gen'
import * as RouteTreeGen from './route-tree-gen'
import * as Saga from '../util/saga'
import * as RPCTypes from '../constants/types/rpc-gen'
import {TypedState} from '../constants/reducer'
import {validateNumber} from '../util/phone-numbers'
import {validateEmailAddress} from '../util/email-address'

const closeTeamBuilding = () => RouteTreeGen.createClearModals()
export type NSAction = {payload: {namespace: TeamBuildingTypes.AllowedNamespace}}
type SearchOrRecAction = {payload: {namespace: TeamBuildingTypes.AllowedNamespace; includeContacts: boolean}}

const apiSearch = async (
  query: string,
  service: TeamBuildingTypes.ServiceIdWithContact,
  maxResults: number,
  includeServicesSummary: boolean,
  impTofuQuery: RPCTypes.ImpTofuQuery | null,
  includeContacts: boolean
): Promise<Array<TeamBuildingTypes.User>> => {
  switch (service) {
    // These services should not be queried through the API.
    // TODO: Y2K-552 change types in this function so it can't be called with
    // invalid services.
    case 'phone':
    case 'contact':
    case 'email':
      return []
  }
  try {
    const results = await RPCTypes.userSearchUserSearchRpcPromise({
      impTofuQuery,
      includeContacts: service === 'keybase' && includeContacts,
      includeServicesSummary,
      maxResults,
      query,
      service,
    })
    return (results || []).reduce<Array<TeamBuildingTypes.User>>((arr, r) => {
      const u = Constants.parseRawResultToUser(r, service)
      u && arr.push(u)
      return arr
    }, [])
  } catch (err) {
    logger.error(`Error in searching for ${query} on ${service}. ${err.message}`)
    return []
  }
}

function* searchResultCounts(state: TypedState, {payload: {namespace}}: NSAction) {
  const teamBuildingState = state[namespace].teamBuilding
  const {teamBuildingSearchQuery, teamBuildingSelectedService} = teamBuildingState
  const teamBuildingSearchLimit = 11 // Hard coded since this happens for background tabs

  if (teamBuildingSearchQuery === '') {
    return
  }

  // Filter on `services` so we only get what's searchable through API.
  // Also filter out if we already have that result cached.
  const servicesToSearch = Constants.services
    .filter(s => s !== teamBuildingSelectedService && !['contact', 'phone', 'email'].includes(s))
    .filter(s => !teamBuildingState.teamBuildingSearchResults.hasIn([teamBuildingSearchQuery, s]))

  const isStillInSameQuery = (state: TypedState): boolean => {
    const teamBuildingState = state[namespace].teamBuilding

    return (
      teamBuildingState.teamBuildingSearchQuery === teamBuildingSearchQuery &&
      teamBuildingState.teamBuildingSelectedService === teamBuildingSelectedService
    )
  }

  // Defer so we aren't conflicting with the main search
  yield Saga.callUntyped(Saga.delay, 100)

  // Change this to control how many requests are in flight at a time
  const parallelRequestsCount = 2

  // Channel to interact with workers. Initial buffer size to handle all the messages we'll put
  // + 1 because we'll put the END message at the end when we close
  const serviceChannel = yield Saga.callUntyped(
    Saga.channel,
    Saga.buffers.expanding(servicesToSearch.length + 1)
  )
  servicesToSearch.forEach(service => serviceChannel.put(service))
  // After the workers pull all the services they can stop
  serviceChannel.close()

  for (let i = 0; i < parallelRequestsCount; i++) {
    yield Saga.spawn(function*() {
      // The loop will exit when we run out of services
      while (true) {
        const service = yield Saga.take(serviceChannel)
        // if we aren't in the same query, let's stop
        if (!isStillInSameQuery(yield* Saga.selectState())) {
          break
        }
        // TODO what happens if this fails?
        const users: Saga.RPCPromiseType<typeof apiSearch> = yield apiSearch(
          teamBuildingSearchQuery,
          service,
          teamBuildingSearchLimit,
          true,
          null,
          false
        )

        yield Saga.put(
          TeamBuildingGen.createSearchResultsLoaded({
            namespace,
            query: teamBuildingSearchQuery,
            service,
            users,
          })
        )
      }
    })
  }
}

const makeImpTofuQuery = (query: string, region: string | null): RPCTypes.ImpTofuQuery | null => {
  const phoneNumber = validateNumber(query, region)
  if (phoneNumber.valid) {
    return {
      phone: phoneNumber.e164,
      t: RPCTypes.ImpTofuSearchType.phone,
    }
  } else if (validateEmailAddress(query)) {
    return {
      email: query,
      t: RPCTypes.ImpTofuSearchType.email,
    }
  }
  return null
}

const search = async (state: TypedState, {payload: {namespace, includeContacts}}: SearchOrRecAction) => {
  const {teamBuildingSearchQuery, teamBuildingSelectedService, teamBuildingSearchLimit} = state[
    namespace
  ].teamBuilding
  // We can only ask the api for at most 100 results
  if (teamBuildingSearchLimit > 100) {
    logger.info('ignoring search request with a limit over 100')
    return false
  }

  const query = teamBuildingSearchQuery
  let impTofuQuery: RPCTypes.ImpTofuQuery | null = null
  if (teamBuildingSelectedService === 'keybase') {
    const userRegion = state.settings.contacts.userCountryCode
    impTofuQuery = makeImpTofuQuery(query, userRegion)
  }

  const users = await apiSearch(
    query,
    teamBuildingSelectedService,
    teamBuildingSearchLimit,
    true,
    impTofuQuery,
    includeContacts
  )
  return TeamBuildingGen.createSearchResultsLoaded({
    namespace,
    query,
    service: teamBuildingSelectedService,
    users,
  })
}

const fetchUserRecs = async (
  state: TypedState,
  {payload: {namespace, includeContacts}}: SearchOrRecAction
) => {
  try {
    const [_suggestionRes, _contactRes] = await Promise.all([
      RPCTypes.userInterestingPeopleRpcPromise({maxUsers: 50}),
      includeContacts
        ? RPCTypes.contactsGetContactsForUserRecommendationsRpcPromise()
        : Promise.resolve([] as RPCTypes.ProcessedContact[]),
    ])
    const suggestionRes = _suggestionRes || []
    const contactRes = _contactRes || []
    const contacts = contactRes.map(Constants.contactToUser)
    let suggestions = suggestionRes.map(Constants.interestingPersonToUser)
    const expectingContacts = state.settings.contacts.importEnabled && includeContacts
    if (expectingContacts) {
      suggestions = suggestions.slice(0, 10)
    }
    return TeamBuildingGen.createFetchedUserRecs({namespace, users: suggestions.concat(contacts)})
  } catch (_) {
    logger.error(`Error in fetching recs`)
    return TeamBuildingGen.createFetchedUserRecs({namespace, users: []})
  }
}

async function searchEmailAddress(state: TypedState, {payload: {namespace}}: SearchOrRecAction) {
  const query = state[namespace].teamBuilding.teamBuildingEmailSearchQuery
  const impTofuQuery = makeImpTofuQuery(query, null)

  const users = await apiSearch(query, 'keybase', 1, true, impTofuQuery, false)
  return TeamBuildingGen.createSearchEmailAddressResultLoaded({
    namespace,
    query,
    user: users[0],
  })
}

export function filterForNs<S, A, L, R>(
  namespace: TeamBuildingTypes.AllowedNamespace,
  fn: (s: S, a: A & NSAction, l: L) => R
) {
  return (s, a, l) => {
    if (a && a.payload && a.payload.namespace === namespace) {
      return fn(s, a, l)
    }
    return undefined
  }
}

function filterGenForNs<S, A, L>(
  namespace: TeamBuildingTypes.AllowedNamespace,
  fn: (s: S, a: A & NSAction, l: L) => Iterable<any>
) {
  return function*(s, a, l) {
    if (a && a.payload && a.payload.namespace === namespace) {
      yield* fn(s, a, l)
    }
  }
}

export default function* commonSagas(
  namespace: TeamBuildingTypes.AllowedNamespace
): Saga.SagaGenerator<any, any> {
  yield* Saga.chainAction2(TeamBuildingGen.search, filterForNs(namespace, search))
  yield* Saga.chainAction2(TeamBuildingGen.fetchUserRecs, filterForNs(namespace, fetchUserRecs))
  yield* Saga.chainGenerator<TeamBuildingGen.SearchPayload>(
    TeamBuildingGen.search,
    filterGenForNs(namespace, searchResultCounts)
  )
  yield* Saga.chainAction2(TeamBuildingGen.searchEmailAddress, filterForNs(namespace, searchEmailAddress))
  // Navigation, before creating
  yield* Saga.chainAction2(
    [TeamBuildingGen.cancelTeamBuilding, TeamBuildingGen.finishedTeamBuilding],
    filterForNs(namespace, closeTeamBuilding)
  )
}
