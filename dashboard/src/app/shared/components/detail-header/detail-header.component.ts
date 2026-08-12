import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatIcon} from '@angular/material/icon';

@Component({
  selector: 'app-detail-header',
  templateUrl: './detail-header.component.html',
  styleUrls: ['./detail-header.component.scss'],
  imports: [MatIcon],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class DetailHeaderComponent {
  @Input({required: true}) icon = '';
  @Input({required: true}) title = '';
  @Input() titleHref = '';
  @Input() supertitle = '';
  @Input() description = '';
}
