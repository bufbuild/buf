// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bufgen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginConfig_ParseRemoteHostName(t *testing.T) {
	host, err := parseCuratedRemoteHostName("buf.build/protocolbuffers/go:v1.28.1")
	require.NoError(t, err)
	require.Equal(
		t,
		"buf.build",
		host,
	)
	host, err = parseCuratedRemoteHostName("buf.build/protocolbuffers/go")
	require.NoError(t, err)
	require.Equal(
		t,
		"buf.build",
		host,
	)
	host, err = parseLegacyRemoteHostName("buf.build/protocolbuffers/plugins/go:v1.28.1-1")
	require.NoError(t, err)
	require.Equal(
		t,
		"buf.build",
		host,
	)
	host, err = parseLegacyRemoteHostName("buf.build/protocolbuffers/plugins/go")
	require.NoError(t, err)
	require.Equal(
		t,
		"buf.build",
		host,
	)
}
